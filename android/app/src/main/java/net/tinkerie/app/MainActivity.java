// The Tinkerie launcher window: the first thing you see, with a tile for
// every app. Created by nrmluvdev for Tinkerie.
package net.tinkerie.app;

import android.Manifest;
import android.content.Context;
import android.content.Intent;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;

public class MainActivity extends WebActivity {
    static volatile MainActivity instance;
    static volatile boolean visible;
    final Handler h = new Handler(Looper.getMainLooper());

    @Override
    protected void onCreate(Bundle state) {
        super.onCreate(state);
        instance = this;
        setTitleAndTask("Tinkerie");
        showOverlay("Starting Tinkerie…", null, null);
        if (Build.VERSION.SDK_INT >= 33 && checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != android.content.pm.PackageManager.PERMISSION_GRANTED) {
            requestPermissions(new String[]{Manifest.permission.POST_NOTIFICATIONS}, 21); // the "running" notice with its Quit button
        }
        TinkerieService s = TinkerieService.instance;
        if (s != null && s.launcherUrl != null) load(s.launcherUrl);
        else waitForLauncher(0);
    }

    // The launcher tells the service its address when it starts; if that
    // takes a while, nudge it.
    void waitForLauncher(final int tries) {
        h.postDelayed(() -> {
            if (pageUrl != null && !pageFailed) return;
            TinkerieService s = TinkerieService.instance;
            if (s != null && s.launcherUrl != null) {
                load(s.launcherUrl);
                return;
            }
            if (s != null) s.ensureLauncher();
            if (tries == 20) showOverlay("Tinkerie is taking a while to start…", "Try again", v -> restart());
            if (tries < 120) waitForLauncher(tries + 1);
        }, 500);
    }

    void restart() {
        TinkerieService s = TinkerieService.instance;
        if (s != null) {
            s.killGo();
            s.ensureLauncher();
        } else TinkerieService.start(this);
        showOverlay("Starting Tinkerie…", null, null);
        pageUrl = null;
        waitForLauncher(0);
    }

    static void show(Context c, String url) {
        MainActivity m = instance;
        if (m != null && !m.isFinishing()) {
            if (!url.equals(m.pageUrl) || m.pageFailed) m.load(url);
            if (!visible) c.startActivity(new Intent(c, MainActivity.class).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_REORDER_TO_FRONT));
            return;
        }
        // nothing on screen yet: Android only lets us open a window if Tinkerie is in front
        if (WebActivity.anyVisible() || c instanceof android.app.Activity) {
            c.startActivity(new Intent(c, MainActivity.class).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK));
        }
    }

    @Override
    protected void onNewIntent(Intent intent) {
        super.onNewIntent(intent);
        TinkerieService.start(this); // e.g. a board was plugged in
    }

    @Override
    protected void onResume() {
        super.onResume();
        visible = true;
        TinkerieService s = TinkerieService.instance;
        if (s != null) s.ensureLauncher();
    }

    @Override
    protected void onPause() {
        visible = false;
        super.onPause();
    }

    @Override
    protected void onDestroy() {
        if (instance == this) instance = null;
        super.onDestroy();
    }

    @Override
    void leave() {
        moveTaskToBack(true);
    }

    @Override
    void pageGone() {
        // the launcher exits when it thinks nobody is looking; start it again
        showOverlay("Starting Tinkerie…", null, null);
        pageUrl = null;
        TinkerieService s = TinkerieService.instance;
        if (s != null) s.ensureLauncher();
        waitForLauncher(0);
    }
}
