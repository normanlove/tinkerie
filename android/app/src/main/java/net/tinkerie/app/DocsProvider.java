// "Tinkerie" in Android's Files app: the Documents folder the apps save into
// (Wirie boards, Flashie downloads, Probie captures, Meterie logs …), so
// people can open, share, copy and delete them like any other files.
// Created by nrmluvdev for Tinkerie.
package net.tinkerie.app;

import android.content.Context;
import android.content.Intent;
import android.database.Cursor;
import android.database.MatrixCursor;
import android.net.Uri;
import android.os.CancellationSignal;
import android.os.ParcelFileDescriptor;
import android.provider.DocumentsContract;
import android.provider.DocumentsContract.Document;
import android.provider.DocumentsContract.Root;
import android.provider.DocumentsProvider;
import android.webkit.MimeTypeMap;

import java.io.File;
import java.io.FileNotFoundException;

public class DocsProvider extends DocumentsProvider {
    static final String AUTHORITY = "net.tinkerie.app.documents";
    static final String ROOT = "root";

    static final String[] ROOT_COLS = {Root.COLUMN_ROOT_ID, Root.COLUMN_FLAGS, Root.COLUMN_ICON, Root.COLUMN_TITLE,
            Root.COLUMN_SUMMARY, Root.COLUMN_DOCUMENT_ID, Root.COLUMN_AVAILABLE_BYTES};
    static final String[] DOC_COLS = {Document.COLUMN_DOCUMENT_ID, Document.COLUMN_MIME_TYPE, Document.COLUMN_DISPLAY_NAME,
            Document.COLUMN_LAST_MODIFIED, Document.COLUMN_FLAGS, Document.COLUMN_SIZE};

    File base() {
        File d = new File(TinkerieService.homeDir(getContext()), "Documents");
        d.mkdirs();
        return d;
    }

    @Override
    public boolean onCreate() {
        return true;
    }

    String idOf(File f) {
        String b = base().getAbsolutePath(), p = f.getAbsolutePath();
        return p.equals(b) ? ROOT : ROOT + p.substring(b.length());
    }

    File fileOf(String id) throws FileNotFoundException {
        if (!id.equals(ROOT) && !id.startsWith(ROOT + "/")) throw new FileNotFoundException(id);
        File f = new File(base(), id.substring(ROOT.length()));
        try { // stay inside Tinkerie's folder
            if (!f.getCanonicalPath().startsWith(base().getCanonicalPath())) throw new FileNotFoundException(id);
        } catch (java.io.IOException e) {
            throw new FileNotFoundException(id);
        }
        return f;
    }

    static String mimeOf(File f) {
        if (f.isDirectory()) return Document.MIME_TYPE_DIR;
        String n = f.getName(), ext = n.lastIndexOf('.') >= 0 ? n.substring(n.lastIndexOf('.') + 1).toLowerCase() : "";
        String m = MimeTypeMap.getSingleton().getMimeTypeFromExtension(ext);
        return m != null ? m : "application/octet-stream";
    }

    void row(MatrixCursor c, File f) {
        int flags = Document.FLAG_SUPPORTS_DELETE | Document.FLAG_SUPPORTS_RENAME;
        if (f.isDirectory()) flags |= Document.FLAG_DIR_SUPPORTS_CREATE;
        else flags |= Document.FLAG_SUPPORTS_WRITE;
        String name = f.equals(base()) ? "Tinkerie" : f.getName();
        c.newRow().add(Document.COLUMN_DOCUMENT_ID, idOf(f)).add(Document.COLUMN_MIME_TYPE, mimeOf(f))
                .add(Document.COLUMN_DISPLAY_NAME, name).add(Document.COLUMN_LAST_MODIFIED, f.lastModified())
                .add(Document.COLUMN_FLAGS, flags).add(Document.COLUMN_SIZE, f.isDirectory() ? null : f.length());
    }

    @Override
    public Cursor queryRoots(String[] projection) {
        MatrixCursor c = new MatrixCursor(projection != null ? projection : ROOT_COLS);
        c.newRow().add(Root.COLUMN_ROOT_ID, ROOT).add(Root.COLUMN_DOCUMENT_ID, ROOT).add(Root.COLUMN_TITLE, "Tinkerie")
                .add(Root.COLUMN_SUMMARY, "Files from Tinkerie's apps").add(Root.COLUMN_ICON, R.mipmap.ic_launcher)
                .add(Root.COLUMN_FLAGS, Root.FLAG_SUPPORTS_CREATE | Root.FLAG_LOCAL_ONLY | Root.FLAG_SUPPORTS_IS_CHILD)
                .add(Root.COLUMN_AVAILABLE_BYTES, base().getFreeSpace());
        return c;
    }

    @Override
    public Cursor queryDocument(String id, String[] projection) throws FileNotFoundException {
        MatrixCursor c = new MatrixCursor(projection != null ? projection : DOC_COLS);
        row(c, fileOf(id));
        return c;
    }

    @Override
    public Cursor queryChildDocuments(String parent, String[] projection, String sort) throws FileNotFoundException {
        MatrixCursor c = new MatrixCursor(projection != null ? projection : DOC_COLS);
        File[] kids = fileOf(parent).listFiles();
        if (kids != null) for (File f : kids) if (!f.getName().startsWith(".")) row(c, f);
        c.setNotificationUri(getContext().getContentResolver(), DocumentsContract.buildChildDocumentsUri(AUTHORITY, parent));
        return c;
    }

    @Override
    public ParcelFileDescriptor openDocument(String id, String mode, CancellationSignal signal) throws FileNotFoundException {
        return ParcelFileDescriptor.open(fileOf(id), ParcelFileDescriptor.parseMode(mode));
    }

    @Override
    public String createDocument(String parent, String mime, String name) throws FileNotFoundException {
        File dir = fileOf(parent), f = new File(dir, name.replace('/', '_'));
        for (int i = 2; f.exists(); i++) f = new File(dir, name + " (" + i + ")");
        try {
            if (Document.MIME_TYPE_DIR.equals(mime) ? !f.mkdirs() : !f.createNewFile()) throw new FileNotFoundException(name);
        } catch (java.io.IOException e) {
            throw new FileNotFoundException(e.getMessage());
        }
        changed(parent);
        return idOf(f);
    }

    @Override
    public void deleteDocument(String id) throws FileNotFoundException {
        File f = fileOf(id);
        if (id.equals(ROOT)) throw new FileNotFoundException("can't delete the Tinkerie folder");
        delete(f);
        changed(idOf(f.getParentFile()));
    }

    static void delete(File f) {
        File[] kids = f.listFiles();
        if (kids != null) for (File k : kids) delete(k);
        f.delete();
    }

    @Override
    public String renameDocument(String id, String name) throws FileNotFoundException {
        File f = fileOf(id), to = new File(f.getParentFile(), name.replace('/', '_'));
        if (!f.renameTo(to)) throw new FileNotFoundException("couldn't rename");
        changed(idOf(f.getParentFile()));
        return idOf(to);
    }

    @Override
    public boolean isChildDocument(String parent, String id) {
        return id.startsWith(parent + "/") || parent.equals(ROOT);
    }

    void changed(String parent) {
        getContext().getContentResolver().notifyChange(DocumentsContract.buildChildDocumentsUri(AUTHORITY, parent), null);
    }

    // Opens the Files app at a folder in Tinkerie's documents; false if it can't.
    static boolean showFolder(Context c, String path) {
        File base = new File(TinkerieService.homeDir(c), "Documents"), f = new File(path);
        if (f.isFile()) f = f.getParentFile();
        String bp = base.getAbsolutePath(), fp = f == null ? bp : f.getAbsolutePath();
        String id = fp.startsWith(bp) ? ROOT + fp.substring(bp.length()) : ROOT;
        Uri u = DocumentsContract.buildDocumentUri(AUTHORITY, id);
        Intent i = new Intent(Intent.ACTION_VIEW).setDataAndType(u, Document.MIME_TYPE_DIR)
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_GRANT_READ_URI_PERMISSION);
        try {
            c.startActivity(i);
            return true;
        } catch (Exception e) {
            try { // older Files apps open at the root
                c.startActivity(new Intent(Intent.ACTION_VIEW).setDataAndType(DocumentsContract.buildRootUri(AUTHORITY, ROOT), "vnd.android.document/root")
                        .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK));
                return true;
            } catch (Exception e2) {
                return false;
            }
        }
    }
}
