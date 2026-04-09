<script>
  import '../app.css';
  import { page } from '$app/stores';
  import { sidebarOpen, toasts, dismissToast } from '$lib/stores.js';

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
        <span class="logo-mark"></span>
        <span class="logo-text">Arkheion</span>
      </a>
    </div>

    <ul class="nav-list">
      <li>
        <a href="/" class="nav-item" class:active={isActive('/')} on:click={() => sidebarOpen.set(false)}>
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/>
            <rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/>
          </svg>
          <span class="nav-label">Dashboard</span>
        </a>
      </li>
      <li>
        <a href="/library" class="nav-item" class:active={isActive('/library')} on:click={() => sidebarOpen.set(false)}>
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
          </svg>
          <span class="nav-label">Library</span>
        </a>
      </li>
      <li>
        <a href="/search" class="nav-item" class:active={isActive('/search')} on:click={() => sidebarOpen.set(false)}>
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
          </svg>
          <span class="nav-label">Search</span>
        </a>
      </li>
      <li>
        <a href="/scan" class="nav-item" class:active={isActive('/scan')} on:click={() => sidebarOpen.set(false)}>
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 9V6a2 2 0 0 1 2-2h3"/><path d="M21 9V6a2 2 0 0 0-2-2h-3"/>
            <path d="M3 15v3a2 2 0 0 0 2 2h3"/><path d="M21 15v3a2 2 0 0 1-2 2h-3"/>
            <line x1="7" y1="8" x2="7" y2="16"/><line x1="11" y1="8" x2="11" y2="16"/>
            <line x1="15" y1="8" x2="15" y2="16"/><line x1="17" y1="8" x2="17" y2="16"/>
          </svg>
          <span class="nav-label">Scan</span>
        </a>
      </li>
      <li>
        <a href="/admin" class="nav-item" class:active={isActive('/admin')} on:click={() => sidebarOpen.set(false)}>
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="3"/>
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
          </svg>
          <span class="nav-label">Admin</span>
        </a>
      </li>
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
    <!-- Top bar (mobile only) -->
    <header class="topbar">
      <button class="menu-btn" on:click={() => sidebarOpen.update(v => !v)} aria-label="Toggle menu">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/>
        </svg>
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
      <button class="toast-close" on:click={() => dismissToast(toast.id)}>&#x2715;</button>
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
    width: 220px;
    min-height: 100vh;
    background: var(--color-bg-elevated);
    border-right: 1px solid var(--color-border);
    display: flex;
    flex-direction: column;
    position: fixed;
    top: 0;
    left: 0;
    z-index: 100;
    transition: transform 0.22s ease;
  }

  .sidebar-header {
    padding: 1.375rem 1.25rem 1rem;
    border-bottom: 1px solid var(--color-border);
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 0.625rem;
    text-decoration: none;
  }
  .logo-mark {
    width: 22px;
    height: 22px;
    background: var(--color-primary);
    border-radius: 4px;
    flex-shrink: 0;
  }
  .logo-text {
    font-size: 1rem;
    font-weight: 700;
    color: var(--color-text);
    letter-spacing: 0.03em;
    text-transform: uppercase;
    font-size: 0.9rem;
  }

  .nav-list {
    list-style: none;
    padding: 0.875rem 0.75rem;
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 0.625rem;
    padding: 0.5rem 0.75rem;
    border-radius: var(--radius);
    color: var(--color-text-muted);
    text-decoration: none;
    font-size: 0.825rem;
    font-weight: 500;
    transition: all var(--transition);
    border-left: 2px solid transparent;
  }
  .nav-item:hover {
    background: var(--color-bg-card);
    color: var(--color-text);
    border-left-color: var(--color-border-strong);
  }
  .nav-item.active {
    background: rgba(192, 57, 43, 0.08);
    color: var(--color-accent);
    border-left-color: var(--color-primary);
  }
  .nav-icon {
    width: 15px;
    height: 15px;
    flex-shrink: 0;
  }

  .sidebar-footer {
    padding: 0.875rem 1.25rem;
    border-top: 1px solid var(--color-border);
  }
  .version { font-size: 0.675rem; color: var(--color-text-dim); letter-spacing: 0.04em; }

  /* Mobile overlay */
  .sidebar-overlay {
    display: none;
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    z-index: 99;
  }

  /* Main area */
  .main-area {
    flex: 1;
    margin-left: 220px;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  /* Topbar (mobile only) */
  .topbar {
    display: none;
    align-items: center;
    gap: 0.875rem;
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
    color: var(--color-text-muted);
    cursor: pointer;
    padding: 0.25rem;
    display: flex;
    align-items: center;
  }
  .menu-btn svg { width: 20px; height: 20px; }
  .topbar-logo {
    font-weight: 700;
    font-size: 0.85rem;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--color-text);
    text-decoration: none;
  }

  .content {
    flex: 1;
    padding: 2rem 2.25rem;
  }

  /* Toast notifications */
  .toast-container {
    position: fixed;
    bottom: 1.25rem;
    right: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.625rem;
    z-index: 1000;
    max-width: 360px;
  }

  .toast {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.75rem 0.875rem;
    border-radius: var(--radius);
    box-shadow: var(--shadow-lg);
    animation: slideIn 0.18s ease;
    font-size: 0.825rem;
  }
  .toast-success { background: #0f2a1a; border: 1px solid rgba(76, 175, 116, 0.3); color: #70c090; }
  .toast-error   { background: #2a0f0f; border: 1px solid rgba(192, 57, 43, 0.4);  color: #e08080; }
  .toast-info    { background: #1a1a1a; border: 1px solid var(--color-border-strong); color: var(--color-text-muted); }
  .toast-warning { background: #231a0a; border: 1px solid rgba(201, 150, 60, 0.3); color: #c9963c; }

  .toast-close {
    background: none;
    border: none;
    color: inherit;
    cursor: pointer;
    font-size: 0.7rem;
    opacity: 0.6;
    flex-shrink: 0;
    padding: 0.125rem;
  }
  .toast-close:hover { opacity: 1; }

  @keyframes slideIn {
    from { transform: translateX(100%); opacity: 0; }
    to   { transform: translateX(0);   opacity: 1; }
  }

  /* Responsive */
  @media (max-width: 768px) {
    .sidebar { transform: translateX(-100%); }
    .sidebar.open { transform: translateX(0); }
    .sidebar-overlay { display: block; }
    .main-area { margin-left: 0; }
    .topbar { display: flex; }
    .content { padding: 1.25rem 1rem; }
  }
</style>
