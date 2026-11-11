// Tinkerie for Android: the service that runs the Go program (the same one
// the desktop uses) and answers its requests over the bridge socket: open an
// app window, open a link or folder, choose a file, list and open USB devices.
// It runs in the foreground so long jobs (flashing, logging) keep going.
// Created by nrmluvdev for Tinkerie.
package net.tinkerie.app;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.Service;
import android.content.Context;
import android.content.Intent;
import android.content.pm.ServiceInfo;
import android.net.LocalServerSocket;
import android.net.LocalSocket;
import android.net.Uri;
import android.os.Build;
import android.os.Handler;
import android.os.IBinder;
import android.os.Looper;
import android.util.Log;
import android.widget.Toast;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileDescriptor;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.security.SecureRandom;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.TimeZone;

public class TinkerieService extends Service {
    static final String TAG = "Tinkerie";
    static final String ACTION_QUIT = "net.tinkerie.app.QUIT";
    static final String CHANNEL = "tinkerie";
    static volatile TinkerieService instance;

    final String bridgeName = "tinkerie-" + Long.toHexString(new SecureRandom().nextLong() & Long.MAX_VALUE);
    final Handler main = new Handler(Looper.getMainLooper());
    LocalServerSocket server;
    Process launcher;
    volatile String launcherUrl;
    File home;

    // ------------------------------------------------------------ lifecycle

    static File homeDir(Context c) {
        File h = new File(c.getFilesDir(), "home");
        h.mkdirs();
        return h;
    }

    static void start(Context c) {
        Intent i = new Intent(c, TinkerieService.class);
        if (Build.VERSION.SDK_INT >= 26) c.startForegroundService(i);
        else c.startService(i);
    }

    @Override
    public void onCreate() {
        super.onCreate();
        instance = this;
        home = new File(getFilesDir(), "home");
        new File(home, "Documents").mkdirs();
        new File(home, "logs").mkdirs();
        goForeground();
        try {
            server = new LocalServerSocket(bridgeName);
            Thread t = new Thread(this::acceptLoop, "tinkerie-bridge");
            t.setDaemon(true);
            t.start();
        } catch (Exception e) {
            Log.e(TAG, "bridge", e);
        }
        ensureLauncher();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        goForeground();
        if (intent != null && ACTION_QUIT.equals(intent.getAction())) {
            quit();
            return START_NOT_STICKY;
        }
        ensureLauncher();
        return START_NOT_STICKY;
    }

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }

    @Override
    public void onDestroy() {
        instance = null;
        killGo();
        try {
            if (server != null) server.close();
        } catch (Exception ignored) {
        }
        super.onDestroy();
    }

    @Override
    public void onTaskRemoved(Intent rootIntent) {
        // swiping the launcher away doesn't stop jobs in other app windows
        super.onTaskRemoved(rootIntent);
    }

    void goForeground() {
        NotificationManager nm = (NotificationManager) getSystemService(NOTIFICATION_SERVICE);
        if (Build.VERSION.SDK_INT >= 26) {
            NotificationChannel ch = new NotificationChannel(CHANNEL, "Tinkerie running", NotificationManager.IMPORTANCE_LOW);
            ch.setDescription("Shown while Tinkerie's apps are running, so jobs like flashing aren't stopped");
            ch.setShowBadge(false);
            nm.createNotificationChannel(ch);
        }
        int imm = Build.VERSION.SDK_INT >= 23 ? PendingIntent.FLAG_IMMUTABLE : 0;
        PendingIntent open = PendingIntent.getActivity(this, 1, new Intent(this, MainActivity.class), imm | PendingIntent.FLAG_UPDATE_CURRENT);
        PendingIntent quit = PendingIntent.getService(this, 2, new Intent(this, TinkerieService.class).setAction(ACTION_QUIT), imm | PendingIntent.FLAG_UPDATE_CURRENT);
        Notification.Builder b = Build.VERSION.SDK_INT >= 26 ? new Notification.Builder(this, CHANNEL) : new Notification.Builder(this);
        b.setSmallIcon(R.drawable.ic_stat)
                .setContentTitle("Tinkerie is running")
                .setContentText("Tap to open. Quit closes every Tinkerie app.")
                .setContentIntent(open)
                .setOngoing(true)
                .addAction(0, "Quit", quit);
        if (Build.VERSION.SDK_INT < 26) b.setPriority(Notification.PRIORITY_LOW);
        Notification n = b.build();
        try {
            if (Build.VERSION.SDK_INT >= 29) startForeground(1, n, ServiceInfo.FOREGROUND_SERVICE_TYPE_CONNECTED_DEVICE);
            else startForeground(1, n);
        } catch (Exception e) {
            Log.w(TAG, "foreground", e); // still works, just without the notification
        }
    }

    void quit() {
        killGo();
        WebActivity.finishAll();
        if (Build.VERSION.SDK_INT >= 24) stopForeground(STOP_FOREGROUND_REMOVE);
        else stopForeground(true);
        stopSelf();
    }

    // ------------------------------------------------------------ the Go program

    String exe() {
        return new File(getApplicationInfo().nativeLibraryDir, "libtinkerie.so").getPath();
    }

    Process spawn(String... args) throws Exception {
        List<String> cmd = new ArrayList<>();
        cmd.add(exe());
        for (String a : args) cmd.add(a);
        ProcessBuilder pb = new ProcessBuilder(cmd);
        pb.directory(home);
        pb.redirectErrorStream(true);
        Map<String, String> env = pb.environment();
        env.put("HOME", home.getPath());
        env.put("TMPDIR", getCacheDir().getPath());
        env.put("TINKERIE_BRIDGE", bridgeName);
        env.put("TINKERIE_TZ", TimeZone.getDefault().getID());
        env.put("TINKERIE_ANDROID_SDK", String.valueOf(Build.VERSION.SDK_INT));
        final Process p = pb.start();
        final String name = args.length > 1 ? args[1] : "launcher";
        Thread t = new Thread(() -> pumpLog(p, name), "tinkerie-log-" + name);
        t.setDaemon(true);
        t.start();
        return p;
    }

    // Each Go process's output goes to Files › Tinkerie › logs, for bug reports.
    void pumpLog(Process p, String name) {
        File f = new File(new File(home, "logs"), name + ".log");
        if (f.length() > (2 << 20)) f.delete();
        try (InputStream in = p.getInputStream(); FileOutputStream out = new FileOutputStream(f, true)) {
            byte[] buf = new byte[8192];
            int n;
            while ((n = in.read(buf)) > 0) {
                out.write(buf, 0, n);
                out.flush();
            }
        } catch (Exception ignored) {
        }
        if ("launcher".equals(name)) {
            synchronized (this) {
                if (launcher == p) {
                    launcher = null;
                    launcherUrl = null;
                }
            }
            // the launcher exits when it thinks its window closed; bring it back if it's on screen
            main.postDelayed(() -> {
                if (MainActivity.visible) ensureLauncher();
            }, 500);
        }
    }

    static boolean alive(Process p) {
        if (p == null) return false;
        try {
            p.exitValue();
            return false;
        } catch (IllegalThreadStateException e) {
            return true;
        }
    }

    synchronized void ensureLauncher() {
        if (alive(launcher)) return;
        try {
            launcherUrl = null;
            launcher = spawn();
        } catch (Exception e) {
            Log.e(TAG, "launcher", e);
            toast("Tinkerie couldn't start: " + e.getMessage());
        }
    }

    // startApp restarts one app (its window found it had closed).
    void startApp(String app) {
        try {
            spawn("--app", app);
        } catch (Exception e) {
            toast("Couldn't start " + app + ": " + e.getMessage());
        }
    }

    // killGo stops every Go process of this app (apps run in their own sessions).
    void killGo() {
        synchronized (this) {
            if (launcher != null) launcher.destroy();
            launcher = null;
            launcherUrl = null;
        }
        int me = android.os.Process.myPid();
        File[] procs = new File("/proc").listFiles();
        if (procs == null) return;
        for (File d : procs) {
            int pid;
            try {
                pid = Integer.parseInt(d.getName());
            } catch (NumberFormatException e) {
                continue;
            }
            if (pid == me) continue;
            try (InputStream in = new java.io.FileInputStream(new File(d, "cmdline"))) {
                byte[] b = new byte[512];
                int n = in.read(b);
                if (n > 0 && new String(b, 0, n, "UTF-8").contains("libtinkerie.so")) android.os.Process.killProcess(pid);
            } catch (Exception ignored) {
            }
        }
    }

    void toast(final String s) {
        main.post(() -> Toast.makeText(getApplicationContext(), s, Toast.LENGTH_LONG).show());
    }

    // ------------------------------------------------------------ the bridge

    void acceptLoop() {
        while (true) {
            final LocalSocket s;
            try {
                s = server.accept();
            } catch (Exception e) {
                return;
            }
            Thread t = new Thread(() -> serve(s), "tinkerie-call");
            t.setDaemon(true);
            t.start();
        }
    }

    void serve(LocalSocket s) {
        Usb.Opened usbOpen = null;
        try {
            if (s.getPeerCredentials().getUid() != android.os.Process.myUid()) return; // only our own Go processes
            BufferedReader r = new BufferedReader(new InputStreamReader(s.getInputStream(), "UTF-8"));
            String line = r.readLine();
            if (line == null) return;
            JSONObject req = new JSONObject(line);
            JSONObject res = new JSONObject();
            FileDescriptor fd = null;
            try {
                switch (req.optString("op")) {
                    case "window":
                        openWindow(req.getString("url"), req.optString("app", "launcher"), req.optString("title"));
                        break;
                    case "browse":
                        browse(req.getString("url"));
                        break;
                    case "folder":
                        if (!DocsProvider.showFolder(this, req.getString("path"))) toast("Saved in Files › Tinkerie");
                        break;
                    case "toast":
                        toast(req.optString("text"));
                        break;
                    case "pick":
                        res.put("path", WebActivity.pickFile(req.optString("title"), req.optJSONArray("exts")));
                        break;
                    case "usblist":
                        res.put("devices", Usb.list(this));
                        break;
                    case "usbopen":
                        usbOpen = Usb.open(this, req.getString("path"));
                        fd = usbOpen.fd.getFileDescriptor();
                        break;
                    default:
                        res.put("error", "Tinkerie for Android can't do \"" + req.optString("op") + "\" yet");
                }
                if (!res.has("error")) res.put("ok", true);
            } catch (Exception e) {
                res = new JSONObject();
                res.put("error", e.getMessage() != null ? e.getMessage() : e.toString());
            }
            if (fd != null) s.setFileDescriptorsForSend(new FileDescriptor[]{fd});
            OutputStream out = s.getOutputStream();
            out.write((res.toString() + "\n").getBytes("UTF-8"));
            out.flush();
        } catch (Exception e) {
            Log.w(TAG, "bridge call", e);
        } finally {
            if (usbOpen != null) usbOpen.close(); // the Go process has its own copy now
            try {
                s.close();
            } catch (Exception ignored) {
            }
        }
    }

    void openWindow(final String url, final String app, final String title) {
        if ("launcher".equals(app)) {
            launcherUrl = url;
            main.post(() -> MainActivity.show(this, url));
            return;
        }
        main.post(() -> AppActivity.open(this, url, app, title));
    }

    void browse(String url) {
        Uri u = url.startsWith("/") ? null : Uri.parse(url);
        if (u == null) { // a file or folder in Tinkerie's storage
            if (!DocsProvider.showFolder(this, url)) toast("Saved in Files › Tinkerie");
            return;
        }
        final Intent i = new Intent(Intent.ACTION_VIEW, u).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
        main.post(() -> {
            try {
                startActivity(i);
            } catch (Exception e) {
                toast("No app here can open " + url);
            }
        });
    }
}
