<template>
  <div class="app">
    <header class="app-header">
      <div class="container header-inner">
        <router-link to="/" class="logo">
          Send
        </router-link>
        <nav class="nav">
          <router-link to="/" class="nav-link">收件</router-link>
          <router-link to="/upload" class="nav-link">上传</router-link>
          <router-link to="/admin" class="nav-link" style="color: var(--color-danger)">管理</router-link>
          <el-tooltip :content="themeTooltip" placement="bottom">
            <button class="theme-toggle" @click="cycleTheme" aria-label="切换主题">
              <el-icon :size="16">
                <Sunny v-if="displayMode === 'light'" />
                <Moon v-else-if="displayMode === 'dark'" />
                <Monitor v-else />
              </el-icon>
            </button>
          </el-tooltip>
        </nav>
      </div>
    </header>
    <main class="container main">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { Sunny, Moon, Monitor } from '@element-plus/icons-vue'

type ThemeMode = 'auto' | 'light' | 'dark'

const STORAGE_KEY = 'send_theme_mode'
const themeMode = ref<ThemeMode>((localStorage.getItem(STORAGE_KEY) as ThemeMode) || 'auto')

// The actual mode being displayed (resolves 'auto' to system preference)
const displayMode = computed<ThemeMode>(() => {
  if (themeMode.value !== 'auto') return themeMode.value
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
})

const themeTooltip = computed(() => {
  const labels: Record<ThemeMode, string> = {
    auto: '跟随系统（当前' + (displayMode.value === 'dark' ? '暗色' : '亮色') + '）',
    light: '亮色模式',
    dark: '暗色模式',
  }
  return labels[themeMode.value]
})

const mq = window.matchMedia('(prefers-color-scheme: dark)')

function applyTheme() {
  const dark = displayMode.value === 'dark'
  document.documentElement.classList.toggle('dark', dark)
  document.documentElement.style.colorScheme = dark ? 'dark' : 'light'
}

function cycleTheme() {
  const order: ThemeMode[] = ['auto', 'light', 'dark']
  const idx = order.indexOf(themeMode.value)
  themeMode.value = order[(idx + 1) % order.length]
  localStorage.setItem(STORAGE_KEY, themeMode.value)
  applyTheme()
}

function onSystemThemeChange() {
  if (themeMode.value === 'auto') applyTheme()
}

onMounted(() => {
  applyTheme()
  mq.addEventListener('change', onSystemThemeChange)
})

onBeforeUnmount(() => {
  mq.removeEventListener('change', onSystemThemeChange)
})
</script>

<style scoped>
.app-header {
  background: var(--bg-card);
  border-bottom: 1px solid var(--border-color);
  position: sticky;
  top: 0;
  z-index: 100;
  transition: background-color 0.3s ease, border-color 0.3s ease;
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
  color: var(--text-primary);
  letter-spacing: -0.5px;
}
.nav {
  display: flex;
  gap: 4px;
  align-items: center;
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
.theme-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  margin-left: 8px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--bg-hover);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
}
.theme-toggle:hover {
  color: var(--text-primary);
  border-color: var(--color-primary);
}
.main {
  padding-top: 32px;
  padding-bottom: 64px;
}
</style>
