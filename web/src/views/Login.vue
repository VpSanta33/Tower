<template>
  <div class="login-container">
    <div class="ambient ambient-one"></div>
    <div class="ambient ambient-two"></div>
    <div class="grid-mask"></div>

    <div class="controls">
      <div class="control-btn" @click="localeStore.toggleLocale">
        <el-icon><Position /></el-icon>
        <span>{{ localeStore.currentLocale === 'zh-CN' ? 'EN' : '中' }}</span>
      </div>
      <div class="control-btn" @click="themeStore.toggleTheme">
        <el-icon v-if="themeStore.isDark"><Sunny /></el-icon>
        <el-icon v-else><Moon /></el-icon>
      </div>
    </div>

    <div class="login-shell">
      <section class="brand-panel">
        <div class="brand-badge">
          <img src="/tower-logo.svg" alt="Tower" />
        </div>
        <div class="brand-copy">
          <p class="eyebrow">Tower Security Platform</p>
          <h1>Tower</h1>
          <p>{{ $t('auth.loginTitle') }}</p>
        </div>
        <div class="signal-card">
          <div class="signal-ring">
            <span></span>
          </div>
          <div>
            <strong>Attack Surface Intelligence</strong>
            <small>Assets · Fingerprints · Vulnerabilities · Workers</small>
          </div>
        </div>
      </section>

      <section class="login-box">
        <div class="login-header">
          <img src="/tower-logo.svg" alt="Tower" />
          <div>
            <h2>{{ $t('auth.login') }}</h2>
            <p>{{ $t('auth.loginTitle') }}</p>
          </div>
        </div>
        <el-form ref="formRef" :model="form" :rules="rules" class="login-form">
          <el-form-item prop="username">
            <el-input
              v-model="form.username"
              :placeholder="$t('auth.username')"
              prefix-icon="User"
              size="large"
            />
          </el-form-item>
          <el-form-item prop="password">
            <el-input
              v-model="form.password"
              type="password"
              :placeholder="$t('auth.password')"
              prefix-icon="Lock"
              size="large"
              show-password
              @keyup.enter="handleLogin"
            />
          </el-form-item>
          <el-form-item>
            <el-button
              type="primary"
              size="large"
              :loading="loading"
              class="login-btn"
              @click="handleLogin"
            >
              {{ $t('auth.login') }}
            </el-button>
          </el-form-item>
        </el-form>
        <div class="login-footer">
          <span></span>
          <p>Distributed Asset Scanning</p>
          <span></span>
        </div>
      </section>
    </div>

    <div class="status-strip">
      <div>
        <strong>API</strong>
        <span>8888</span>
      </div>
      <div>
        <strong>RPC</strong>
        <span>9000</span>
      </div>
      <div>
        <strong>WEB</strong>
        <span>3333</span>
      </div>
    </div>

    <el-dialog
      v-model="resetDialogVisible"
      :title="$t('auth.forceResetTitle', '首次登录请修改密码')"
      width="400px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      :show-close="false"
      destroy-on-close
    >
      <el-form ref="resetFormRef" :model="resetForm" :rules="resetRules" label-width="auto" label-position="top">
        <el-form-item :label="$t('auth.newPassword', '新密码')" prop="newPassword">
          <el-input v-model="resetForm.newPassword" type="password" show-password />
        </el-form-item>
        <el-form-item :label="$t('auth.confirmPassword', '确认密码')" prop="confirmPassword">
          <el-input v-model="resetForm.confirmPassword" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button type="primary" :loading="resetLoading" @click="handleResetSubmit" style="width: 100%;">
          {{ $t('common.confirm', '确认修改进入系统') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { useThemeStore } from '@/stores/theme'
import { useLocaleStore } from '@/stores/locale'
import { Sunny, Moon, Position } from '@element-plus/icons-vue'
import { firstLoginResetPassword } from '@/api/auth'

const router = useRouter()
const { t } = useI18n()
const userStore = useUserStore()
const themeStore = useThemeStore()
const localeStore = useLocaleStore()
const formRef = ref()
const loading = ref(false)

const form = reactive({
  username: '',
  password: ''
})

const rules = computed(() => ({
  username: [{ required: true, message: t('auth.pleaseEnterUsername'), trigger: 'blur' }],
  password: [{ required: true, message: t('auth.pleaseEnterPassword'), trigger: 'blur' }]
}))

// 强制修密码逻辑
const resetDialogVisible = ref(false)
const resetLoading = ref(false)
const resetFormRef = ref()
const resetForm = reactive({
  newPassword: '',
  confirmPassword: ''
})

const resetRules = computed(() => {
  const validatePass2 = (rule, value, callback) => {
    if (value === '') {
      callback(new Error(t('auth.pleaseConfirmPassword', '请再次输入密码')))
    } else if (value !== resetForm.newPassword) {
      callback(new Error(t('auth.passwordMismatch', '两次输入密码不一致')))
    } else {
      callback()
    }
  }
  return {
    newPassword: [
      { required: true, message: t('auth.pleaseEnterNewPassword', '请输入新密码'), trigger: 'blur' },
      { min: 6, message: t('auth.passwordMinLengths', '密码长度不能小于6位'), trigger: 'blur' }
    ],
    confirmPassword: [
      { required: true, validator: validatePass2, trigger: 'blur' }
    ]
  }
})

async function handleLogin() {
  try { await formRef.value.validate() } catch { return }
  loading.value = true
  try {
    const res = await userStore.login(form)
    if (res.code === 0) {
      if (res.needChangePwd) {
        resetDialogVisible.value = true
        return // 中断后续的弹窗和页面跳转，等待密码重置
      }
      ElMessage.success(t('auth.loginSuccess'))
      router.push('/dashboard')
    } else {
      ElMessage.error(res.msg || t('auth.loginFailed'))
    }
  } finally {
    loading.value = false
  }
}

async function handleResetSubmit() {
  try { await resetFormRef.value.validate() } catch { return }
  resetLoading.value = true
  try {
    const res = await firstLoginResetPassword({
      id: userStore.userId,
      newPassword: resetForm.newPassword
    })
    if (res.code === 0) {
      ElMessage.success(t('auth.passwordResetSuccess', '密码修改成功，欢迎进入系统'))
      resetDialogVisible.value = false
      router.push('/dashboard')
    } else {
      ElMessage.error(res.msg || t('auth.passwordResetFailed', '密码修改失败'))
    }
  } catch (error) {
    console.error(error)
    ElMessage.error(t('auth.passwordResetFailed', '请求失败'))
  } finally {
    resetLoading.value = false
  }
}
</script>

<style scoped>
.login-container {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  background:
    radial-gradient(circle at 18% 18%, rgba(34, 211, 238, 0.2), transparent 28%),
    radial-gradient(circle at 82% 12%, rgba(37, 99, 235, 0.22), transparent 32%),
    linear-gradient(135deg, hsl(var(--background)) 0%, hsl(var(--muted)) 100%);
  color: hsl(var(--foreground));
  transition: background 0.3s;
}

.ambient,
.grid-mask {
  position: absolute;
  pointer-events: none;
}

.ambient {
  width: 360px;
  height: 360px;
  border-radius: 999px;
  filter: blur(18px);
  opacity: 0.6;
}

.ambient-one {
  left: -130px;
  top: -110px;
  background: radial-gradient(circle, rgba(34, 211, 238, 0.32), transparent 68%);
}

.ambient-two {
  right: -140px;
  bottom: -130px;
  background: radial-gradient(circle, rgba(37, 99, 235, 0.32), transparent 68%);
}

.grid-mask {
  inset: 0;
  opacity: 0.38;
  background-image:
    linear-gradient(hsl(var(--border) / 0.42) 1px, transparent 1px),
    linear-gradient(90deg, hsl(var(--border) / 0.42) 1px, transparent 1px);
  background-size: 42px 42px;
  mask-image: radial-gradient(circle at center, black 0%, transparent 72%);
}

.controls {
  position: absolute;
  z-index: 3;
  top: 24px;
  right: 28px;
  display: flex;
  gap: 12px;
}

.control-btn {
  cursor: pointer;
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 2px;
  border-radius: 14px;
  background: hsl(var(--card) / 0.78);
  border: 1px solid hsl(var(--border) / 0.88);
  color: hsl(var(--muted-foreground));
  box-shadow: 0 12px 34px rgba(15, 23, 42, 0.12);
  backdrop-filter: blur(16px);
  transition: all 0.25s ease;

  &:hover {
    transform: translateY(-2px);
    border-color: hsl(var(--primary));
    color: hsl(var(--primary));
    box-shadow: 0 18px 42px rgba(37, 99, 235, 0.18);
  }

  .el-icon {
    font-size: 18px;
  }

  span {
    font-size: 12px;
    font-weight: 700;
  }
}

.login-shell {
  position: relative;
  z-index: 2;
  width: min(980px, 100%);
  min-height: 560px;
  display: grid;
  grid-template-columns: minmax(0, 1.08fr) 430px;
  overflow: hidden;
  border: 1px solid hsl(var(--border) / 0.72);
  border-radius: 28px;
  background: hsl(var(--card) / 0.76);
  box-shadow: 0 28px 90px rgba(15, 23, 42, 0.18);
  backdrop-filter: blur(22px);
}

.brand-panel {
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 46px;
  overflow: hidden;
  background:
    linear-gradient(145deg, rgba(7, 17, 31, 0.96), rgba(15, 35, 64, 0.88)),
    radial-gradient(circle at 68% 30%, rgba(103, 232, 249, 0.22), transparent 35%);
  color: #f8fdff;

  &::before {
    content: '';
    position: absolute;
    inset: 28px;
    border: 1px solid rgba(103, 232, 249, 0.16);
    border-radius: 24px;
  }

  &::after {
    content: '';
    position: absolute;
    right: -78px;
    top: 64px;
    width: 260px;
    height: 260px;
    border-radius: 999px;
    border: 1px solid rgba(103, 232, 249, 0.28);
    box-shadow: inset 0 0 36px rgba(56, 189, 248, 0.14);
  }
}

.brand-badge,
.brand-copy,
.signal-card {
  position: relative;
  z-index: 1;
}

.brand-badge {
  width: 84px;
  height: 84px;
  display: grid;
  place-items: center;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(103, 232, 249, 0.22);
  box-shadow: 0 20px 60px rgba(34, 211, 238, 0.18);

  img {
    width: 66px;
    height: 66px;
  }
}

.brand-copy {
  max-width: 410px;

  .eyebrow {
    margin: 0 0 16px;
    color: #67e8f9;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.2em;
    text-transform: uppercase;
  }

  h1 {
    margin: 0;
    font-size: clamp(54px, 8vw, 82px);
    line-height: 0.95;
    letter-spacing: 0.05em;
  }

  p:not(.eyebrow) {
    margin: 22px 0 0;
    color: rgba(248, 253, 255, 0.68);
    font-size: 16px;
    line-height: 1.8;
  }
}

.signal-card {
  width: min(100%, 390px);
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 18px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.07);
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 18px 54px rgba(0, 0, 0, 0.18);

  strong,
  small {
    display: block;
  }

  strong {
    margin-bottom: 6px;
    font-size: 14px;
  }

  small {
    color: rgba(248, 253, 255, 0.58);
    line-height: 1.6;
  }
}

.signal-ring {
  width: 54px;
  height: 54px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border-radius: 999px;
  border: 1px solid rgba(103, 232, 249, 0.44);
  background: radial-gradient(circle, rgba(103, 232, 249, 0.18), transparent 70%);

  span {
    width: 14px;
    height: 14px;
    border-radius: 999px;
    background: #67e8f9;
    box-shadow: 0 0 22px #67e8f9;
  }
}

.login-box {
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 48px 44px;
  background: hsl(var(--card) / 0.9);
}

.login-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 34px;

  img {
    width: 56px;
    height: 56px;
    filter: drop-shadow(0 14px 24px rgba(37, 99, 235, 0.22));
  }

  h2 {
    margin: 0 0 6px;
    color: hsl(var(--foreground));
    font-size: 28px;
    font-weight: 700;
    letter-spacing: 0.08em;
  }

  p {
    margin: 0;
    color: hsl(var(--muted-foreground));
    font-size: 14px;
  }
}

.login-form {
  :deep(.el-form-item) {
    margin-bottom: 22px;
  }

  :deep(.el-input__wrapper) {
    min-height: 48px;
    background: hsl(var(--background) / 0.78);
    border: 1px solid hsl(var(--border));
    box-shadow: none;
    border-radius: 14px;
    transition: all 0.22s ease;

    &:hover {
      border-color: hsl(var(--primary) / 0.45);
    }

    &.is-focus {
      border-color: hsl(var(--primary));
      box-shadow: 0 0 0 4px hsl(var(--primary) / 0.12);
    }
  }

  :deep(.el-input__inner) {
    color: hsl(var(--foreground));

    &::placeholder {
      color: hsl(var(--muted-foreground));
    }
  }

  :deep(.el-input__prefix) {
    color: hsl(var(--muted-foreground));
  }

  .login-btn {
    width: 100%;
    height: 48px;
    overflow: hidden;
    border: none;
    border-radius: 14px;
    color: #fff;
    background: linear-gradient(135deg, #22d3ee 0%, #2563eb 100%);
    box-shadow: 0 16px 34px rgba(37, 99, 235, 0.3);
    font-size: 16px;
    font-weight: 700;
    letter-spacing: 0.12em;
    transition: all 0.24s ease;

    &:hover {
      transform: translateY(-1px);
      box-shadow: 0 20px 42px rgba(37, 99, 235, 0.38);
    }
  }
}

.login-footer {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 10px;
  color: hsl(var(--muted-foreground));
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;

  span {
    height: 1px;
    flex: 1;
    background: hsl(var(--border));
  }

  p {
    margin: 0;
    white-space: nowrap;
  }
}

.status-strip {
  position: absolute;
  z-index: 2;
  left: 50%;
  bottom: 22px;
  transform: translateX(-50%);
  display: flex;
  gap: 10px;
  padding: 8px;
  border-radius: 999px;
  background: hsl(var(--card) / 0.68);
  border: 1px solid hsl(var(--border) / 0.72);
  backdrop-filter: blur(14px);

  div {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 7px 12px;
    border-radius: 999px;
    color: hsl(var(--muted-foreground));
    font-size: 12px;
  }

  strong {
    color: hsl(var(--foreground));
    font-size: 11px;
    letter-spacing: 0.12em;
  }
}

@media (max-width: 860px) {
  .login-container {
    align-items: flex-start;
    padding-top: 86px;
  }

  .login-shell {
    min-height: auto;
    grid-template-columns: 1fr;
  }

  .brand-panel {
    min-height: 260px;
    padding: 30px;
  }

  .brand-copy h1 {
    font-size: 48px;
  }

  .signal-card {
    display: none;
  }

  .login-box {
    padding: 34px 26px;
  }

  .status-strip {
    display: none;
  }
}

@media (max-width: 520px) {
  .login-container {
    padding: 76px 14px 24px;
  }

  .controls {
    top: 18px;
    right: 16px;
  }

  .brand-panel {
    min-height: 220px;
  }

  .brand-badge {
    width: 68px;
    height: 68px;

    img {
      width: 52px;
      height: 52px;
    }
  }

  .login-header {
    align-items: flex-start;

    img {
      width: 48px;
      height: 48px;
    }
  }
}
</style>
