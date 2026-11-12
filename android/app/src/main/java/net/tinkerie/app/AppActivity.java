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

    android.widget.TextView layoutButton;

    @Override
    protected void onCreate(Bundle state) {
        super.onCreate(state);
        take(getIntent());
        if (getResources().getConfiguration().screenWidthDp < 900) addLayoutButton();
    }

    // Phones: switch between the app's desktop layout (pinch to zoom) and
    // fitting it to the phone's width. Remembered for each app.
    void addLayoutButton() {
        layoutButton = new android.widget.TextView(this);
        layoutButton.setTextColor(0xFFE8ECF3);
        layoutButton.setTextSize(12);
        layoutButton.setPadding(24, 10, 24, 10);
        android.graphics.drawable.GradientDrawable bg = new android.graphics.drawable.GradientDrawable();
        bg.setColor(0xCC1A2230);
        bg.setCornerRadius(40);
        bg.setStroke(2, 0x552DD4BF);
        layoutButton.setBackground(bg);
        layoutButton.setAlpha(0.75f);
        updateLayoutButton();
        layoutButton.setOnClickListener(v -> {
            prefs().edit().putBoolean("desktop." + app, !desktop()).apply();
            updateLayoutButton();
            if (pageUrl != null) {
                web.evaluateJavascript("window.__tk=0", null);
                web.reload();
            }
        });
        android.widget.FrameLayout.LayoutParams lp = new android.widget.FrameLayout.LayoutParams(
                android.view.ViewGroup.LayoutParams.WRAP_CONTENT, android.view.ViewGroup.LayoutParams.WRAP_CONTENT,
                android.view.Gravity.BOTTOM | android.view.Gravity.END);
        lp.setMargins(0, 0, 16, 16);
        root.addView(layoutButton, lp);
    }

    void updateLayoutButton() {
        if (layoutButton != null) layoutButton.setText(desktop() ? "⇔ Fit to phone" : "⇔ Desktop layout");
    }

    android.content.SharedPreferences prefs() {
        return getSharedPreferences("layout", MODE_PRIVATE);
    }

    boolean desktop() {
        return prefs().getBoolean("desktop." + app, !"codie".equals(app)); // Codie already suits a phone
    }

    @Override
    boolean wantsDesktopWidth() {
        return desktop();
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
            case 47825: return "fixie";
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
