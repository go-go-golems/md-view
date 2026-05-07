// Mermaid initialization for md-view.
// Detects ```mermaid code blocks and renders them as SVG diagrams.
(function() {
    // Check if there are any mermaid code blocks on the page
    var mermaidBlocks = document.querySelectorAll('code.language-mermaid');
    if (mermaidBlocks.length === 0) return;

    // Wrap each <code class="language-mermaid"> in a <div class="mermaid">
    // so mermaid.js can find and render them
    mermaidBlocks.forEach(function(codeBlock) {
        var pre = codeBlock.parentElement;
        if (pre && pre.tagName === 'PRE') {
            var div = document.createElement('div');
            div.className = 'mermaid';
            div.textContent = codeBlock.textContent;
            pre.parentNode.replaceChild(div, pre);
        }
    });

    // Initialize mermaid with the current theme
    var isDark = document.documentElement.getAttribute('data-theme') === 'dark';
    mermaid.initialize({
        startOnLoad: true,
        theme: isDark ? 'dark' : 'default',
        securityLevel: 'loose'
    });
})();
