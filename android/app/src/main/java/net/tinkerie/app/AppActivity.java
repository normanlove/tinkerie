// One Tinkerie app (Wirie, Flashie, Scopie …) in its own window. Each app
// has its own entry in Recents, like separate windows on a computer.
// Created by nrmluvdev for Tinkerie.
package net.tinkerie.app;

import android.app.Activity;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.widget.Toast;

public class AppActivity extends WebActivity {
    String app;

    static String nice(String app) {
        if (app == null || app.isEmpty()) return "Tinkerie";
        return Character.toUpperCase(app.charAt(0)) + app.substring(1);
    }

    static void open(Context c, String url, String app, String title) {
        Intent i = new Intent(c, AppActivity.class);
        i.setData(Uri.parse(url));
        i.putExtra("app", app);
        i.putExtra("title", title == null || title.isEmpty() ? nice(app) : title);
        i.addFlags(Intent.FLAG_ACTIVITY_NEW_DOCUMENT | Intent.FLAG_ACTIVITY_NEW_TASK);
        if (c instanceof Activity || WebActivity.anyVisible()) {
            c.startActivity(i);
            return;
        }
        // Tinkerie isn't on screen, and Android won't let it pop a window up: offer it
        try {
            int f = PendingIntent.FLAG_UPDATE_CURRENT | (Build.VERSION.SDK_INT >= 23 ? PendingIntent.FLAG_IMMUTABLE : 0);
            PendingIntent.getActivity(c, url.hashCode(), i, f).send();
        } catch (Exception e) {
            Toast.makeText(c, "Open Tinkerie to see " + nice(app), Toast.LENGTH_LONG).show();
        }
    }

    @Override
    protected void onCreate(Bundle state) {
        super.onCreate(state);
        take(getIntent());
    }

    @Override
    protected void onNewIntent(Intent intent) {
        super.onNewIntent(intent);
        setIntent(intent);
        take(intent);
    }

    void take(Intent i) {
        Uri u = i.getData();
        String a = i.getStringExtra("app");
        if (a != null && !a.isEmpty()) app = a;
        if (app == null || app.isEmpty()) app = guessApp(u);
        setTitleAndTask(i.getStringExtra("title") != null ? i.getStringExtra("title") : nice(app));
        if (u == null) return;
        if (pageUrl == null || pageFailed || !u.toString().equals(pageUrl)) {
            showOverlay("Opening " + nice(app) + "…", null, null);
            load(u.toString());
        }
    }

    // Apps use fixed ports; when a window opens from a link, the port says which app it is.
    static String guessApp(Uri u) {
        if (u == null) return "";
        switch (u.getPort()) {
            case 47811: return "makie";
            case 47820: return "wirie";
            case 47821: return "tunie";
            case 47822: return "probie";
            case 47824: return "codie";
            case 47831: return "scopie";
            default: return "";
        }
    }

    @Override
    void leave() {
        finishAndRemoveTask();
    }

    @Override
    void pageGone() {
        showOverlay(nice(app) + " has closed.", "Open it again", v -> {
            TinkerieService s = TinkerieService.instance;
            if (s == null) {
                TinkerieService.start(this);
                return;
            }
            if (app != null && !app.isEmpty()) {
                showOverlay("Opening " + nice(app) + "…", null, null);
                pageUrl = null;
                s.startApp(app); // it asks for its window again, which lands here
            } else finishAndRemoveTask();
        });
    }
}
