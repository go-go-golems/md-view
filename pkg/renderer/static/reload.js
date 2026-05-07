// MDSReloader — SSE client for live reload in md-view.
// Opens an EventSource to the /events endpoint and reloads the page on "reload" events.
function MDSReloader(eventsURL) {
    var es = new EventSource(eventsURL);
    es.addEventListener("reload", function() {
        location.reload();
    });
    es.onerror = function() {
        // Reconnect after 2 seconds
        setTimeout(function() {
            new MDSReloader(eventsURL);
        }, 2000);
        es.close();
    };
}
