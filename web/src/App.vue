<template>
  <div class="app">
    <header class="app-header">
      <div class="container header-inner">
        <router-link to="/" class="logo">
          Send
        </router-link>
        <nav class="nav">
          <router-link to="/" class="nav-link">上传</router-link>
          <router-link to="/admin" class="nav-link" style="color: var(--color-danger)">管理</router-link>
        </nav>
      </div>
    </header>
    <main class="container main">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.app-header {
  background: var(--bg-card);
  border-bottom: 1px solid var(--border-color);
  position: sticky;
  top: 0;
  z-index: 100;
}
.header-inner {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.logo {
  font-size: 20px;
  font-weight: 700;
  color: #000;
  letter-spacing: -0.5px;
}
.nav {
  display: flex;
  gap: 4px;
}
.nav-link {
  color: var(--text-secondary);
  font-size: 14px;
  padding: 6px 12px;
  border-radius: 4px;
  transition: all 0.15s ease;
}
.nav-link:hover,
.nav-link.router-link-active {
  color: var(--text-primary);
  background: var(--bg-hover);
}
.main {
  padding-top: 32px;
  padding-bottom: 64px;
}
</style>
