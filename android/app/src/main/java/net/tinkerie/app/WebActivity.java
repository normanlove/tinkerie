// A window showing one Tinkerie app's page (served by its Go process on
// 127.0.0.1). It does for the page what a desktop browser would: choose and
// save files, use the microphone, open other apps' windows and web links.
// Created by nrmluvdev for Tinkerie.
package net.tinkerie.app;

import android.Manifest;
import android.annotation.SuppressLint;
import android.app.Activity;
import android.app.ActivityManager;
import android.content.ActivityNotFoundException;
import android.content.Intent;
import android.content.pm.ApplicationInfo;
import android.content.pm.PackageManager;
import android.content.res.Configuration;
import android.database.Cursor;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.graphics.Color;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.os.Message;
import android.provider.OpenableColumns;
import android.util.Base64;
import android.util.Log;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.view.WindowInsets;
import android.webkit.ConsoleMessage;
import android.webkit.JavascriptInterface;
import android.webkit.PermissionRequest;
import android.webkit.URLUtil;
import android.webkit.ValueCallback;
import android.webkit.WebChromeClient;
import android.webkit.WebResourceError;
import android.webkit.WebResourceRequest;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.Button;
import android.widget.FrameLayout;
import android.widget.LinearLayout;
import android.widget.TextView;
import android.widget.Toast;

import org.json.JSONArray;

import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

public abstract class WebActivity extends Activity {
    static final int REQ_CHOOSER = 11, REQ_PICK = 12, REQ_SAVE = 13, REQ_MIC = 14;
    static final List<WebActivity> open = new ArrayList<>();
    static volatile WebActivity top;

    FrameLayout root;
    WebView web;
    LinearLayout overlay;
    TextView overlayText;
    Button overlayButton;
    String pageUrl;
    boolean pageFailed;

    ValueCallback<Uri[]> chooserCallback;
    PickCallback pickCallback;
    File pendingSave;
    PermissionRequest pendingMic;

    interface PickCallback {
        void done(Uri uri);
    }

    // ------------------------------------------------------------ set-up

    @SuppressLint({"SetJavaScriptEnabled", "AddJavascriptInterface"})
    @Override
    protected void onCreate(Bundle state) {
        super.onCreate(state);
        synchronized (open) {
            open.add(this);
        }
        TinkerieService.start(this);
        if ((getApplicationInfo().flags & ApplicationInfo.FLAG_DEBUGGABLE) != 0) WebView.setWebContentsDebuggingEnabled(true);

        root = new FrameLayout(this);
        root.setBackgroundColor(0xFF0E1218);
        web = new WebView(this);
        root.addView(web, new FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));
        buildOverlay();
        setContentView(root);
        fitSystemBars();

        WebSettings s = web.getSettings();
        s.setJavaScriptEnabled(true);
        s.setDomStorageEnabled(true);
        s.setDatabaseEnabled(true);
        s.setMediaPlaybackRequiresUserGesture(false);
        s.setAllowFileAccess(false);
        s.setAllowContentAccess(true);
        s.setTextZoom(100); // the apps size their own text
        s.setSupportZoom(true);
        s.setBuiltInZoomControls(true);
        s.setDisplayZoomControls(false);
        s.setUseWideViewPort(true);
        s.setLoadWithOverviewMode(true);
        s.setSupportMultipleWindows(true);
        s.setJavaScriptCanOpenWindowsAutomatically(true);
        s.setUserAgentString(s.getUserAgentString() + " TinkerieAndroid/" + BuildInfo.version(this));
        web.setBackgroundColor(0xFF0E1218);
        web.addJavascriptInterface(new JsBridge(), "TinkerieAndroid");
        web.setWebViewClient(new Client());
        web.setWebChromeClient(new Chrome());
        web.setDownloadListener((url, ua, disposition, mime, length) -> download(url, disposition, mime));
        checkWebViewVersion();
    }

    // Android 15 draws apps edge to edge: keep the page clear of the bars and keyboard.
    void fitSystemBars() {
        root.setOnApplyWindowInsetsListener((v, ins) -> {
            if (Build.VERSION.SDK_INT >= 30) {
                android.graphics.Insets i = ins.getInsets(WindowInsets.Type.systemBars() | WindowInsets.Type.displayCutout() | WindowInsets.Type.ime());
                v.setPadding(i.left, i.top, i.right, i.bottom);
            } else {
                v.setPadding(ins.getSystemWindowInsetLeft(), ins.getSystemWindowInsetTop(), ins.getSystemWindowInsetRight(), ins.getSystemWindowInsetBottom());
            }
            return ins;
        });
        root.requestApplyInsets();
    }

    void buildOverlay() {
        overlay = new LinearLayout(this);
        overlay.setOrientation(LinearLayout.VERTICAL);
        overlay.setGravity(Gravity.CENTER);
        overlay.setBackgroundColor(0xFF0E1218);
        overlay.setPadding(48, 48, 48, 48);
        overlayText = new TextView(this);
        overlayText.setTextColor(0xFFE8ECF3);
        overlayText.setTextSize(17);
        overlayText.setGravity(Gravity.CENTER);
        overlay.addView(overlayText);
        overlayButton = new Button(this);
        overlayButton.setVisibility(View.GONE);
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        lp.topMargin = 32;
        overlay.addView(overlayButton, lp);
        root.addView(overlay, new FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));
    }

    void showOverlay(String text, String button, View.OnClickListener click) {
        overlayText.setText(text);
        overlay.setVisibility(View.VISIBLE);
        if (button == null) overlayButton.setVisibility(View.GONE);
        else {
            overlayButton.setText(button);
            overlayButton.setOnClickListener(click);
            overlayButton.setVisibility(View.VISIBLE);
        }
    }

    void hideOverlay() {
        overlay.setVisibility(View.GONE);
    }

    void load(String url) {
        pageUrl = url;
        pageFailed = false;
        web.loadUrl(url);
    }

    void setTitleAndTask(String title) {
        setTitle(title);
        Bitmap icon = BitmapFactory.decodeResource(getResources(), R.mipmap.ic_launcher);
        setTaskDescription(new ActivityManager.TaskDescription(title, icon, 0xFF0E1218));
    }

    // The apps' pages were made for computer screens. On phones and small
    // tablets lay them out 1100 px wide and let pinch-zoom do the rest.
    int layoutWidth() {
        Configuration c = getResources().getConfiguration();
        return c.screenWidthDp < 900 ? 1100 : 0;
    }

    // Older Android System WebView versions can't run the apps' pages.
    void checkWebViewVersion() {
        Matcher m = Pattern.compile("Chrome/(\\d+)").matcher(web.getSettings().getUserAgentString());
        if (m.find() && Integer.parseInt(m.group(1)) < 80) {
            Toast.makeText(this, "Please update \"Android System WebView\" (or Chrome) from the Play Store: Tinkerie needs a newer one.", Toast.LENGTH_LONG).show();
        }
    }

    @Override
    protected void onResume() {
        super.onResume();
        top = this;
        web.onResume();
    }

    @Override
    protected void onPause() {
        if (top == this) top = null;
        web.onPause(); // the page's own timers keep running, so jobs carry on
        super.onPause();
    }

    @Override
    protected void onDestroy() {
        synchronized (open) {
            open.remove(this);
        }
        if (top == this) top = null;
        web.destroy();
        super.onDestroy();
    }

    static void finishAll() {
        List<WebActivity> all;
        synchronized (open) {
            all = new ArrayList<>(open);
        }
        for (WebActivity a : all) a.finishAndRemoveTask();
    }

    static boolean anyVisible() {
        return top != null;
    }

    @Override
    public void onBackPressed() {
        if (web.canGoBack()) web.goBack();
        else leave();
    }

    abstract void leave();

    abstract void pageGone(); // the app's Go process has stopped

    // ------------------------------------------------------------ navigation

    static boolean isLocal(Uri u) {
        String h = u.getHost();
        return h != null && (h.equals("127.0.0.1") || h.equals("localhost"));
    }

    // Links: this app's pages stay here, another app's page opens its own
    // window, and anything else goes to the browser.
    boolean route(Uri u) {
        if (u == null) return true;
        String scheme = u.getScheme() == null ? "" : u.getScheme();
        if (scheme.equals("blob") || scheme.equals("data") || scheme.equals("javascript") || scheme.equals("about")) return false;
        if (isLocal(u) && pageUrl != null) {
            Uri here = Uri.parse(pageUrl);
            if (u.getPort() == here.getPort()) return false;
            AppActivity.open(this, u.toString(), "", "Tinkerie");
            return true;
        }
        try {
            startActivity(new Intent(Intent.ACTION_VIEW, u));
        } catch (ActivityNotFoundException e) {
            Toast.makeText(this, "No app here can open " + u, Toast.LENGTH_LONG).show();
        }
        return true;
    }

    class Client extends WebViewClient {
        @Override
        public boolean shouldOverrideUrlLoading(WebView v, String url) {
            return route(Uri.parse(url));
        }

        @Override
        public void onPageStarted(WebView v, String url, Bitmap favicon) {
            injectShim();
        }

        @Override
        public void onPageFinished(WebView v, String url) {
            injectShim();
            if (!pageFailed) hideOverlay();
        }

        @Override
        public void onReceivedError(WebView v, int code, String desc, String url) {
            if (url != null && url.equals(v.getUrl()) || url != null && url.equals(pageUrl)) {
                pageFailed = true;
                pageGone();
            }
        }

        @Override
        public void onReceivedError(WebView v, WebResourceRequest req, WebResourceError err) {
            if (req.isForMainFrame()) {
                pageFailed = true;
                pageGone();
            }
        }
    }

    class Chrome extends WebChromeClient {
        @Override
        public boolean onShowFileChooser(WebView v, ValueCallback<Uri[]> cb, FileChooserParams p) {
            if (chooserCallback != null) chooserCallback.onReceiveValue(null);
            chooserCallback = cb;
            try {
                Intent i = p.createIntent();
                i.addCategory(Intent.CATEGORY_OPENABLE);
                if (p.getMode() == FileChooserParams.MODE_OPEN_MULTIPLE) i.putExtra(Intent.EXTRA_ALLOW_MULTIPLE, true);
                if (i.getType() == null || i.getType().isEmpty() || i.getType().startsWith(".")) i.setType("*/*"); // .uf2, .kicad_pcb … have no MIME type
                startActivityForResult(i, REQ_CHOOSER);
            } catch (Exception e) {
                chooserCallback = null;
                return false;
            }
            return true;
        }

        @Override
        public void onPermissionRequest(final PermissionRequest r) {
            runOnUiThread(() -> {
                boolean mic = false;
                for (String res : r.getResources()) if (PermissionRequest.RESOURCE_AUDIO_CAPTURE.equals(res)) mic = true;
                if (!mic) {
                    r.deny();
                    return;
                }
                if (Build.VERSION.SDK_INT < 23 || checkSelfPermission(Manifest.permission.RECORD_AUDIO) == PackageManager.PERMISSION_GRANTED) {
                    r.grant(new String[]{PermissionRequest.RESOURCE_AUDIO_CAPTURE});
                } else {
                    pendingMic = r;
                    requestPermissions(new String[]{Manifest.permission.RECORD_AUDIO}, REQ_MIC);
                }
            });
        }

        @Override
        public boolean onCreateWindow(WebView v, boolean dialog, boolean gesture, Message result) {
            // window.open(): find out where it was going, then route it
            WebView probe = new WebView(WebActivity.this);
            probe.setWebViewClient(new WebViewClient() {
                @Override
                public boolean shouldOverrideUrlLoading(WebView pv, String url) {
                    if (!route(Uri.parse(url))) AppActivity.open(WebActivity.this, url, "", "Tinkerie");
                    pv.destroy();
                    return true;
                }
            });
            WebView.WebViewTransport t = (WebView.WebViewTransport) result.obj;
            t.setWebView(probe);
            result.sendToTarget();
            return true;
        }

        @Override
        public void onReceivedTitle(WebView v, String title) {
            if (title != null && !title.isEmpty() && !title.startsWith("127.0.0.1")) setTitleAndTask(title);
        }

        @Override
        public boolean onConsoleMessage(ConsoleMessage m) {
            if (m.messageLevel() == ConsoleMessage.MessageLevel.ERROR) Log.w("TinkerieJS", m.message() + " @" + m.sourceId() + ":" + m.lineNumber());
            return true;
        }
    }

    // ------------------------------------------------------------ page helpers

    // Runs in every page: saves downloads made in the page (Save buttons that
    // build a file in JavaScript), and gives phones a desktop-width layout.
    void injectShim() {
        String js = "(function(){if(window.__tk)return;window.__tk=1;"
                + "var W=" + layoutWidth() + ";"
                + "function vp(){if(!W||!document.head)return;var m=document.querySelector('meta[name=viewport]');"
                + "if(!m){m=document.createElement('meta');m.name='viewport';document.head.appendChild(m);}"
                + "m.content='width='+W;}vp();document.addEventListener('DOMContentLoaded',vp);"
                + "var rv=URL.revokeObjectURL;URL.revokeObjectURL=function(u){setTimeout(function(){rv.call(URL,u)},60000)};"
                + "function save(h,n){fetch(h).then(function(r){return r.blob()}).then(function(b){var f=new FileReader();"
                + "f.onload=function(){var s=f.result;TinkerieAndroid.saveFile(n||'download',b.type||'application/octet-stream',s.substring(s.indexOf(',')+1))};"
                + "f.readAsDataURL(b)}).catch(function(e){TinkerieAndroid.toast('Saving failed: '+e)})}"
                + "function grab(a){if(!a||!a.href||a.getAttribute('download')===null)return false;"
                + "if(!/^(blob:|data:|https?:\\/\\/(127\\.0\\.0\\.1|localhost))/.test(a.href))return false;"
                + "save(a.href,a.getAttribute('download')||decodeURIComponent(a.href.split('/').pop().split('?')[0]));return true}"
                + "var oc=HTMLAnchorElement.prototype.click;HTMLAnchorElement.prototype.click=function(){if(grab(this))return;return oc.apply(this,arguments)};"
                + "document.addEventListener('click',function(e){var a=e.target&&e.target.closest&&e.target.closest('a[download]');"
                + "if(a&&grab(a)){e.preventDefault();e.stopPropagation()}},true);"
                + "})();";
        web.evaluateJavascript(js, null);
    }

    class JsBridge {
        @JavascriptInterface
        public void saveFile(String name, String mime, String b64) {
            try {
                File f = new File(getCacheDir(), "save-" + System.nanoTime());
                try (FileOutputStream o = new FileOutputStream(f)) {
                    o.write(Base64.decode(b64, Base64.DEFAULT));
                }
                runOnUiThread(() -> askWhereToSave(f, name, mime));
            } catch (Exception e) {
                toast("Saving failed: " + e.getMessage());
            }
        }

        @JavascriptInterface
        public void toast(String s) {
            runOnUiThread(() -> Toast.makeText(WebActivity.this, s, Toast.LENGTH_LONG).show());
        }

        @JavascriptInterface
        public String version() {
            return BuildInfo.version(WebActivity.this);
        }
    }

    // A file the app's server sends as a download.
    void download(final String url, final String disposition, final String mime) {
        Uri u = Uri.parse(url);
        if (!isLocal(u)) {
            route(u);
            return;
        }
        final String name = URLUtil.guessFileName(url, disposition, mime);
        new Thread(() -> {
            try {
                HttpURLConnection c = (HttpURLConnection) new URL(url).openConnection();
                File f = new File(getCacheDir(), "save-" + System.nanoTime());
                try (InputStream in = c.getInputStream(); FileOutputStream o = new FileOutputStream(f)) {
                    copy(in, o);
                }
                runOnUiThread(() -> askWhereToSave(f, name, mime));
            } catch (Exception e) {
                runOnUiThread(() -> Toast.makeText(this, "Download failed: " + e.getMessage(), Toast.LENGTH_LONG).show());
            }
        }).start();
    }

    // Like "Save as…": the person chooses where the file goes.
    void askWhereToSave(File f, String name, String mime) {
        if (pendingSave != null) pendingSave.delete();
        pendingSave = f;
        Intent i = new Intent(Intent.ACTION_CREATE_DOCUMENT);
        i.addCategory(Intent.CATEGORY_OPENABLE);
        i.setType(mime == null || mime.isEmpty() ? "application/octet-stream" : mime);
        i.putExtra(Intent.EXTRA_TITLE, name);
        try {
            startActivityForResult(i, REQ_SAVE);
        } catch (ActivityNotFoundException e) { // no picker: keep it in Tinkerie's own files
            File dest = new File(new File(TinkerieService.homeDir(this), "Documents/Downloads"), name);
            dest.getParentFile().mkdirs();
            f.renameTo(dest);
            pendingSave = null;
            Toast.makeText(this, "Saved in Files › Tinkerie › Downloads", Toast.LENGTH_LONG).show();
        }
    }

    // Called from the bridge thread: let the person choose a file, copy it in.
    static String pickFile(String title, JSONArray exts) throws Exception {
        final WebActivity a = top;
        if (a == null) throw new Exception("Open the Tinkerie window to choose a file");
        final CountDownLatch done = new CountDownLatch(1);
        final Uri[] got = new Uri[1];
        a.runOnUiThread(() -> {
            a.pickCallback = uri -> {
                got[0] = uri;
                done.countDown();
            };
            Intent i = new Intent(Intent.ACTION_OPEN_DOCUMENT);
            i.addCategory(Intent.CATEGORY_OPENABLE);
            i.setType("*/*");
            try {
                a.startActivityForResult(i, REQ_PICK);
            } catch (Exception e) {
                a.pickCallback = null;
                done.countDown();
            }
        });
        done.await(30, TimeUnit.MINUTES);
        if (got[0] == null) return "";
        String name = displayName(a, got[0]);
        File dir = new File(TinkerieService.homeDir(a), "picked");
        dir.mkdirs();
        File dest = new File(dir, name);
        try (InputStream in = a.getContentResolver().openInputStream(got[0]); FileOutputStream o = new FileOutputStream(dest)) {
            copy(in, o);
        }
        return dest.getPath();
    }

    static String displayName(Activity a, Uri u) {
        String name = null;
        try (Cursor c = a.getContentResolver().query(u, new String[]{OpenableColumns.DISPLAY_NAME}, null, null, null)) {
            if (c != null && c.moveToFirst()) name = c.getString(0);
        } catch (Exception ignored) {
        }
        if (name == null || name.isEmpty()) name = u.getLastPathSegment();
        if (name == null || name.isEmpty()) name = "file";
        return name.replace('/', '_');
    }

    static void copy(InputStream in, OutputStream out) throws java.io.IOException {
        byte[] b = new byte[65536];
        int n;
        while ((n = in.read(b)) > 0) out.write(b, 0, n);
    }

    @Override
    protected void onActivityResult(int req, int result, Intent data) {
        super.onActivityResult(req, result, data);
        if (req == REQ_CHOOSER && chooserCallback != null) {
            chooserCallback.onReceiveValue(WebChromeClient.FileChooserParams.parseResult(result, data));
            chooserCallback = null;
        } else if (req == REQ_PICK && pickCallback != null) {
            PickCallback cb = pickCallback;
            pickCallback = null;
            cb.done(result == RESULT_OK && data != null ? data.getData() : null);
        } else if (req == REQ_SAVE && pendingSave != null) {
            File f = pendingSave;
            pendingSave = null;
            if (result == RESULT_OK && data != null && data.getData() != null) {
                try (InputStream in = new FileInputStream(f); OutputStream out = getContentResolver().openOutputStream(data.getData())) {
                    copy(in, out);
                    Toast.makeText(this, "Saved", Toast.LENGTH_SHORT).show();
                } catch (Exception e) {
                    Toast.makeText(this, "Saving failed: " + e.getMessage(), Toast.LENGTH_LONG).show();
                }
            }
            f.delete();
        }
    }

    @Override
    public void onRequestPermissionsResult(int req, String[] perms, int[] results) {
        if (req == REQ_MIC && pendingMic != null) {
            if (results.length > 0 && results[0] == PackageManager.PERMISSION_GRANTED) pendingMic.grant(new String[]{PermissionRequest.RESOURCE_AUDIO_CAPTURE});
            else pendingMic.deny();
            pendingMic = null;
        }
    }

    void toast(String s) {
        Toast.makeText(this, s, Toast.LENGTH_LONG).show();
    }
}
