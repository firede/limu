// The examples' theme switch and their "Copy source" buttons.

const themes = ['kohu', 'kohu-dark', 'default'] as const;
type Theme = typeof themes[number];

// Until a theme is chosen, the diagrams follow the page: Kohu in its light theme, KohuDark in
// its dark one. A choice of the theme they would follow is forgotten, as the page's own
// switch forgets a choice of the system's theme.
function setupDiagramThemeSwitch() {
  const scope = document.querySelector<HTMLElement>('[data-diagram-scope]');
  const buttons = document.querySelectorAll<HTMLButtonElement>('[data-diagram-theme-option]');
  if (!scope || buttons.length === 0) return;
  const system = window.matchMedia('(prefers-color-scheme: dark)');
  const root = document.documentElement;
  const followed = (): Theme => ((root.dataset.theme ? root.dataset.theme === 'dark' : system.matches) ? 'kohu-dark' : 'kohu');
  const current = (): Theme => (themes as readonly string[]).includes(scope.dataset.diagramTheme ?? '') ? scope.dataset.diagramTheme as Theme : followed();
  const show = () => {
    const theme = current();
    buttons.forEach((button) => button.setAttribute('aria-pressed', String(button.dataset.diagramThemeOption === theme)));
  };
  buttons.forEach((button) => {
    button.addEventListener('click', () => {
      const theme = button.dataset.diagramThemeOption as Theme;
      if (theme === followed()) {
        delete scope.dataset.diagramTheme;
        try { localStorage.removeItem('limu-diagram-theme'); } catch {}
      } else {
        scope.dataset.diagramTheme = theme;
        try { localStorage.setItem('limu-diagram-theme', theme); } catch {}
      }
      show();
    });
  });
  system.addEventListener('change', show);
  new MutationObserver(show).observe(root, { attributes: true, attributeFilter: ['data-theme'] });
  show();
}

const checkIcon = '<svg aria-hidden="true" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"></path></svg>';
const timers = new WeakMap<HTMLButtonElement, number>();

function setupCopySource() {
  document.addEventListener('click', async (event) => {
    const button = (event.target as Element).closest<HTMLButtonElement>('[data-copy-source]');
    if (!button) return;
    // The button as it was before it was pressed, to go back to.
    button.dataset.idle ??= button.innerHTML;
    try {
      await navigator.clipboard.writeText(button.dataset.copySource ?? '');
      button.dataset.copied = '';
      button.innerHTML = `${checkIcon}<span>Copied</span>`;
    } catch {
      button.dataset.copyError = '';
      button.innerHTML = `${button.querySelector('svg')?.outerHTML ?? ''}<span>Copy failed</span>`;
    }
    const previous = timers.get(button);
    if (previous) window.clearTimeout(previous);
    timers.set(button, window.setTimeout(() => {
      timers.delete(button);
      delete button.dataset.copied;
      delete button.dataset.copyError;
      button.innerHTML = button.dataset.idle!;
    }, 2000));
  });
}

setupDiagramThemeSwitch();
setupCopySource();
