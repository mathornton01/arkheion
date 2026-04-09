<script>
  import '../app.css';
  import { page } from '$app/stores';
  import { sidebarOpen, toasts, dismissToast } from '$lib/stores.js';

  const navItems = [
    { href: '/', label: 'Dashboard', icon: '🏠' },
    { href: '/library', label: 'Library', icon: '📚' },
    { href: '/search', label: 'Search', icon: '🔍' },
    { href: '/scan', label: 'Scan', icon: '📷' },
    { href: '/admin', label: 'Admin', icon: '⚙️' }
  ];

  $: currentPath = $page.url.pathname;

  function isActive(href) {
    if (href === '/') return currentPath === '/';
    return currentPath.startsWith(href);
  }
</script>

<div class="app-shell">
  <!-- Sidebar -->
  <nav class="sidebar" class:open={$sidebarOpen}>
    <div class="sidebar-header">
      <a href="/" class="logo">
        <span class="logo-icon">📖</span>
        <span class="logo-text">Arkheion</span>
      </a>
    </div>

    <ul class="nav-list">
      {#each navItems as item}
        <li>
          <a
            href={item.href}
            class="nav-item"
            class:active={isActive(item.href)}
            on:click={() => sidebarOpen.set(false)}
          >
            <span class="nav-icon">{item.icon}</span>
            <span class="nav-label">{item.label}</span>
          </a>
        </li>
      {/each}
    </ul>

    <div class="sidebar-footer">
      <span class="version">v1.0.0</span>
    </div>
  </nav>

  <!-- Mobile sidebar overlay -->
  {#if $sidebarOpen}
    <div class="sidebar-overlay" on:click={() => sidebarOpen.set(false)}></div>
  {/if}

  <!-- Main content area -->
  <div class="main-area">
    <!-- Top bar (mobile) -->
    <header class="topbar">
      <button class="menu-btn" on:click={() => sidebarOpen.update(v => !v)} aria-label="Toggle menu">
        ☰
      </button>
      <a href="/" class="topbar-logo">Arkheion</a>
    </header>

    <!-- Page content -->
    <main class="content">
      <slot />
    </main>
  </div>
</div>

<!-- Toast notifications -->
<div class="toast-container">
  {#each $toasts as toast (toast.id)}
    <div class="toast toast-{toast.type}" role="alert">
      <span class="toast-message">{toast.message}</span>
      <button class="toast-close" on:click={() => dismissToast(toast.id)}>✕</button>
    </div>
  {/each}
</div>

<style>
  .app-shell {
    display: flex;
    min-height: 100vh;
  }

  /* Sidebar */
  .sidebar {
    width: 240px;
    min-height: 100vh;
    background: var(--color-bg-elevated);
    border-right: 1px solid var(--color-border);
    display: flex;
    flex-direction: column;
    position: fixed;
    top: 0;
    left: 0;
    z-index: 100;
    transition: transform 0.25s ease;
  }

  .sidebar-header {
    padding: 1.5rem 1.25rem 1rem;
    border-bottom: 1px solid var(--color-border);
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    text-decoration: none;
  }
  .logo-icon { font-size: 1.5rem; }
  .logo-text {
    font-size: 1.125rem;
    font-weight: 700;
    color: var(--color-text);
    letter-spacing: -0.02em;
  }

  .nav-list {
    list-style: none;
    padding: 1rem 0.75rem;
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.625rem 0.875rem;
    border-radius: var(--radius);
    color: var(--color-text-muted);
    text-decoration: none;
    font-size: 0.9rem;
    font-weight: 500;
    transition: all var(--transition);
  }
  .nav-item:hover {
    background: var(--color-bg-card);
    color: var(--color-text);
  }
  .nav-item.active {
    background: rgba(124, 106, 247, 0.15);
    color: var(--color-primary);
  }
  .nav-icon { font-size: 1rem; width: 1.25rem; text-align: center; }

  .sidebar-footer {
    padding: 1rem 1.25rem;
    border-top: 1px solid var(--color-border);
  }
  .version { font-size: 0.7rem; color: var(--color-text-dim); }

  /* Mobile overlay */
  .sidebar-overlay {
    display: none;
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    z-index: 99;
  }

  /* Main area */
  .main-area {
    flex: 1;
    margin-left: 240px;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  /* Topbar (mobile only) */
  .topbar {
    display: none;
    align-items: center;
    gap: 1rem;
    padding: 0.75rem 1rem;
    background: var(--color-bg-elevated);
    border-bottom: 1px solid var(--color-border);
    position: sticky;
    top: 0;
    z-index: 50;
  }
  .menu-btn {
    background: none;
    border: none;
    color: var(--color-text);
    font-size: 1.25rem;
    cursor: pointer;
    padding: 0.25rem;
  }
  .topbar-logo {
    font-weight: 700;
    color: var(--color-text);
    text-decoration: none;
  }

  .content {
    flex: 1;
    padding: 2rem;
  }

  /* Toast notifications */
  .toast-container {
    position: fixed;
    bottom: 1.5rem;
    right: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    z-index: 1000;
    max-width: 380px;
  }

  .toast {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.875rem 1rem;
    border-radius: var(--radius);
    box-shadow: var(--shadow-lg);
    animation: slideIn 0.2s ease;
    font-size: 0.875rem;
  }
  .toast-success { background: #1a3a2a; border: 1px solid rgba(58, 247, 138, 0.3); color: #60ff9a; }
  .toast-error   { background: #3a1a1a; border: 1px solid rgba(247, 80, 58, 0.3);  color: #ff9080; }
  .toast-info    { background: #1a1a3a; border: 1px solid rgba(124, 106, 247, 0.3); color: #a89aff; }
  .toast-warning { background: #3a2a1a; border: 1px solid rgba(247, 226, 58, 0.3); color: #ffd060; }

  .toast-close {
    background: none;
    border: none;
    color: inherit;
    cursor: pointer;
    font-size: 0.75rem;
    opacity: 0.7;
    flex-shrink: 0;
  }
  .toast-close:hover { opacity: 1; }

  @keyframes slideIn {
    from { transform: translateX(100%); opacity: 0; }
    to   { transform: translateX(0);   opacity: 1; }
  }

  /* Responsive — mobile */
  @media (max-width: 768px) {
    .sidebar {
      transform: translateX(-100%);
    }
    .sidebar.open {
      transform: translateX(0);
    }
    .sidebar-overlay {
      display: block;
    }
    .main-area {
      margin-left: 0;
    }
    .topbar {
      display: flex;
    }
    .content {
      padding: 1rem;
    }
  }
</style>
