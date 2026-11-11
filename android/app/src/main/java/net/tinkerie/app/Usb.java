// USB for Tinkerie's Go apps on Android: they can't see /dev/bus/usb, so this
// lists devices with Android's USB manager and, once the person allows it,
// opens one and hands its file descriptor to the Go process. The Go side then
// uses the kernel's usbfs on it exactly as on Linux.
// Created by nrmluvdev for Tinkerie.
package net.tinkerie.app;

import android.app.PendingIntent;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.hardware.usb.UsbConstants;
import android.hardware.usb.UsbDevice;
import android.hardware.usb.UsbDeviceConnection;
import android.hardware.usb.UsbEndpoint;
import android.hardware.usb.UsbInterface;
import android.hardware.usb.UsbManager;
import android.os.Build;
import android.os.ParcelFileDescriptor;
import android.util.Base64;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.ByteArrayOutputStream;
import java.util.HashSet;
import java.util.Set;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

class Usb {
    static final String ACTION_PERMISSION = "net.tinkerie.app.USB_PERMISSION";

    static JSONArray list(Context c) throws Exception {
        UsbManager m = (UsbManager) c.getSystemService(Context.USB_SERVICE);
        JSONArray out = new JSONArray();
        if (m == null) return out;
        for (UsbDevice d : m.getDeviceList().values()) {
            JSONObject o = new JSONObject();
            o.put("Path", d.getDeviceName());
            o.put("VID", d.getVendorId());
            o.put("PID", d.getProductId());
            o.put("BCD", bcd(d));
            o.put("Manufacturer", nz(d.getManufacturerName()));
            o.put("Product", nz(d.getProductName()));
            String serial = "";
            try {
                serial = nz(d.getSerialNumber()); // needs permission on Android 10+
            } catch (SecurityException ignored) {
            }
            o.put("Serial", serial);
            o.put("Accessible", m.hasPermission(d));
            JSONArray ifs = new JSONArray();
            Set<Integer> seen = new HashSet<>();
            for (int i = 0; i < d.getInterfaceCount(); i++) {
                UsbInterface it = d.getInterface(i);
                if (!seen.add(it.getId())) continue; // first alternate setting only
                JSONObject j = new JSONObject();
                j.put("Number", it.getId());
                j.put("Class", it.getInterfaceClass());
                j.put("Sub", it.getInterfaceSubclass());
                ByteArrayOutputStream in = new ByteArrayOutputStream(), outEp = new ByteArrayOutputStream();
                for (int e = 0; e < it.getEndpointCount(); e++) {
                    UsbEndpoint ep = it.getEndpoint(e);
                    if (ep.getType() != UsbConstants.USB_ENDPOINT_XFER_BULK) continue;
                    if (ep.getDirection() == UsbConstants.USB_DIR_IN) in.write(ep.getAddress());
                    else outEp.write(ep.getAddress());
                }
                // Go reads []uint8 from JSON as base64
                j.put("BulkIn", Base64.encodeToString(in.toByteArray(), Base64.NO_WRAP));
                j.put("BulkOut", Base64.encodeToString(outEp.toByteArray(), Base64.NO_WRAP));
                ifs.put(j);
            }
            o.put("Interfaces", ifs);
            out.put(o);
        }
        return out;
    }

    static String nz(String s) {
        return s == null ? "" : s.trim();
    }

    // bcdDevice, which tells FTDI chips apart. Android 6+ reports it as "x.yy".
    static int bcd(UsbDevice d) {
        if (Build.VERSION.SDK_INT < 23) return 0;
        try {
            String[] v = d.getVersion().split("\\.");
            return (Integer.parseInt(v[0], 16) << 8) | Integer.parseInt(v[1], 16);
        } catch (Exception e) {
            return 0;
        }
    }

    static class Opened {
        UsbDeviceConnection conn;
        ParcelFileDescriptor fd;

        void close() {
            try {
                if (fd != null) fd.close();
            } catch (Exception ignored) {
            }
            if (conn != null) conn.close();
        }
    }

    static Opened open(Context c, String path) throws Exception {
        UsbManager m = (UsbManager) c.getSystemService(Context.USB_SERVICE);
        UsbDevice d = m == null ? null : m.getDeviceList().get(path);
        if (d == null) throw new Exception("that USB device isn't plugged in any more");
        if (!m.hasPermission(d)) {
            final CountDownLatch done = new CountDownLatch(1);
            BroadcastReceiver r = new BroadcastReceiver() {
                @Override
                public void onReceive(Context ctx, Intent i) {
                    done.countDown();
                }
            };
            IntentFilter f = new IntentFilter(ACTION_PERMISSION);
            if (Build.VERSION.SDK_INT >= 33) c.registerReceiver(r, f, Context.RECEIVER_NOT_EXPORTED);
            else c.registerReceiver(r, f);
            int flags = PendingIntent.FLAG_UPDATE_CURRENT;
            if (Build.VERSION.SDK_INT >= 31) flags |= PendingIntent.FLAG_MUTABLE; // Android fills in the answer
            PendingIntent pi = PendingIntent.getBroadcast(c, 7, new Intent(ACTION_PERMISSION).setPackage(c.getPackageName()), flags);
            m.requestPermission(d, pi);
            try {
                done.await(3, TimeUnit.MINUTES);
            } finally {
                try {
                    c.unregisterReceiver(r);
                } catch (Exception ignored) {
                }
            }
            if (!m.hasPermission(d)) throw new Exception("Tinkerie wasn't allowed to use " + label(d) + ". Unplug it and plug it in again to be asked again");
        }
        Opened o = new Opened();
        o.conn = m.openDevice(d);
        if (o.conn == null) throw new Exception("Android couldn't open " + label(d));
        o.fd = ParcelFileDescriptor.fromFd(o.conn.getFileDescriptor()); // a copy for the Go process
        return o;
    }

    static String label(UsbDevice d) {
        String p = nz(d.getProductName());
        return p.isEmpty() ? String.format("the USB device %04x:%04x", d.getVendorId(), d.getProductId()) : p;
    }
}
