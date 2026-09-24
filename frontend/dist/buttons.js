// buttons.js — fixed-position toolbar buttons for the md-view desktop frontend.
//
// In the daemon/HTTP model these were `remarkable-button.js` + `toolbar-buttons.js`
// running once as IIFEs and POSTing to /upload-remarkable + /raw. In the Wails
// model they call BOUND GO METHODS (window.go.main.App.*) instead of fetch, and
// re-bind after every content swap (MDSInitButtons), because the current file
// path changes per open.
//
// Exposes:
//   window.MDSInitButtons() — (re)create the reMarkable + copy-path + download
//                              buttons for the currently open file.
(function () {
    var tabletIcon = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="2" width="16" height="20" rx="2"/><line x1="12" y1="18" x2="12" y2="18.01"/></svg>';
    var spinnerIcon = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>';
    var checkIcon = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>';
    var errorIcon = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>';
    var clipboardIcon = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="8" y="2" width="12" height="14" rx="2"/><path d="M16 4H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10"/></svg>';
    var downloadIcon = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>';

    function el(cls, title, icon) {
        var b = document.createElement('button');
        b.className = 'md-view-toolbar-btn ' + cls;
        b.title = title;
        b.setAttribute('aria-label', title);
        b.innerHTML = icon;
        return b;
    }

    function toast(html, duration, cls) {
        var t = document.createElement('div');
        t.className = 'md-view-remarkable-toast' + (cls ? ' ' + cls : '');
        t.innerHTML = html;
        document.body.appendChild(t);
        if (duration) setTimeout(function () { t.remove(); }, duration);
        else setTimeout(function () { t.remove(); }, 6000);
    }

    // (Re)build the button row for the current file. Idempotent: removes any
    // existing row first so it can be called after every file-open / live reload.
    window.MDSInitButtons = function () {
        var old = document.getElementById('md-view-button-row');
        if (old) old.remove();

        var App = window['go'] && window['go']['main'] && window['go']['main']['App'];
        if (!App) return;

        // Ask the backend for the current file (set on open). If none, no buttons.
        // (Called async below; we build the row once we know the path.)
        App.GetCurrentFile().then(function (path) {
            if (!path) return;
            buildRow(path, App);
        });
    };

    function buildRow(filePath, App) {
        var row = document.createElement('div');
        row.id = 'md-view-button-row';
        // Layout (fixed position below the app toolbar, buttons in a flex
        // row) is defined in style.css under #md-view-button-row: the
        // ui.css per-button fixed offsets apply only to the legacy
        // full-page renderer, not here.;

        // --- Copy path (clip path) ---
        var copyBtn = el('md-view-copy-path-btn', 'Copy file path to clipboard', clipboardIcon);
        copyBtn.addEventListener('click', function () {
            window.MDSCopyText(filePath).then(function () {
                copyBtn.innerHTML = checkIcon;
                copyBtn.classList.add('md-view-toolbar-btn-success');
                setTimeout(function () { copyBtn.innerHTML = clipboardIcon; copyBtn.classList.remove('md-view-toolbar-btn-success'); }, 2000);
            }).catch(function (e) { toast('✗ Copy failed: ' + e, 5000, 'md-view-remarkable-toast-error'); });
        });

        // --- Copy entire article (markdown source) ---
        var articleIcon = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>';
        var articleBtn = el('md-view-copy-article-btn', 'Copy entire article to clipboard', articleIcon);
        articleBtn.addEventListener('click', function () {
            App.RawFile(filePath).then(function (data) {
                // Wails JSON-marshals Go []byte as a base64 string; decode it.
                var text;
                if (typeof data === 'string') {
                    var bin = atob(data);
                    var bytes = new Uint8Array(bin.length);
                    for (var i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
                    text = new TextDecoder().decode(bytes);
                } else if (data && data.byteLength !== undefined) {
                    text = new TextDecoder().decode(data);
                } else {
                    text = String(data);
                }
                return window.MDSCopyText(text);
            }).then(function () {
                articleBtn.innerHTML = checkIcon;
                articleBtn.classList.add('md-view-toolbar-btn-success');
                setTimeout(function () { articleBtn.innerHTML = articleIcon; articleBtn.classList.remove('md-view-toolbar-btn-success'); }, 2000);
            }).catch(function (e) {
                toast('✗ Copy failed: ' + e, 5000, 'md-view-remarkable-toast-error');
            });
        });

        // --- Download markdown (native save dialog) ---
        var dlBtn = el('md-view-download-btn', 'Download markdown', downloadIcon);
        dlBtn.addEventListener('click', function () {
            App.DownloadMarkdown(filePath).then(function (dest) {
                if (dest) {
                    dlBtn.innerHTML = checkIcon;
                    dlBtn.classList.add('md-view-toolbar-btn-success');
                    setTimeout(function () { dlBtn.innerHTML = downloadIcon; dlBtn.classList.remove('md-view-toolbar-btn-success'); }, 2000);
                }
            }).catch(function (e) { toast('✗ Download failed: ' + e, 5000, 'md-view-remarkable-toast-error'); });
        });

        // --- reMarkable upload ---
        var rmBtn = el('md-view-remarkable-btn', 'Upload to reMarkable', tabletIcon);
        rmBtn.addEventListener('click', function () {
            if (rmBtn.disabled) return;
            rmBtn.disabled = true;
            rmBtn.innerHTML = spinnerIcon;
            rmBtn.classList.add('md-view-remarkable-btn-loading');
            toast('Uploading to reMarkable…', 0, 'md-view-remarkable-toast-loading');
            App.UploadToRemarkable(filePath).then(function (msg) {
                rmBtn.disabled = false;
                rmBtn.innerHTML = checkIcon;
                rmBtn.classList.add('md-view-remarkable-btn-success');
                toast('✓ ' + (msg || 'Uploaded successfully'), 4000, 'md-view-remarkable-toast-success');
                setTimeout(function () { rmBtn.innerHTML = tabletIcon; rmBtn.classList.remove('md-view-remarkable-btn-success'); }, 3000);
            }).catch(function (e) {
                rmBtn.disabled = false;
                rmBtn.innerHTML = errorIcon;
                rmBtn.classList.add('md-view-remarkable-btn-error');
                toast('✗ ' + e, 6000, 'md-view-remarkable-toast-error');
                setTimeout(function () { rmBtn.innerHTML = tabletIcon; rmBtn.classList.remove('md-view-remarkable-btn-error'); }, 5000);
            });
        });

        row.appendChild(copyBtn);
        row.appendChild(articleBtn);
        row.appendChild(dlBtn);
        row.appendChild(rmBtn);
        document.body.appendChild(row);
    }
})();
