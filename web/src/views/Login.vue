<template>
  <div class="login-wrap">
    <div class="login-card card pop">
      <div class="brand-logo" style="width: 52px; height: 52px; font-size: 27px; margin: 0 auto 16px">🍦</div>
      <h1 style="text-align: center; font-size: 21px">frp 管理面板</h1>
      <p class="muted" style="text-align: center; margin: 4px 0 22px; font-size: 13px">请输入面板密码登录</p>

      <div class="field">
        <label>密码</label>
        <div class="pw-wrap">
          <input
            ref="pwInput"
            class="input"
            :type="show ? 'text' : 'password'"
            v-model="password"
            placeholder="面板密码"
            @keyup.enter="login"
          />
          <button class="pw-eye" type="button" @click="show = !show">
            <Icon :name="show ? 'eye-off' : 'eye'" style="width: 17px; height: 17px" />
          </button>
        </div>
      </div>

      <button class="btn btn-primary btn-lg btn-block mt" :disabled="busy" @click="login">
        <Icon v-if="busy" name="loader" class="spin" />
        <Icon v-else name="lock" />
        登录
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import Icon from '../components/Icon.vue'
import { api } from '../api.js'
import { refreshSession } from '../router.js'
import { toast } from '../composables/useToast.js'

const router = useRouter()
const route = useRoute()
const password = ref('')
const show = ref(false)
const busy = ref(false)
const pwInput = ref(null)

onMounted(() => pwInput.value?.focus())

async function login() {
  if (!password.value) return
  busy.value = true
  try {
    await api.login(password.value)
    await refreshSession()
    const redirect = route.query.redirect
    router.push(redirect ? String(redirect) : { name: 'dashboard' })
  } catch (e) {
    toast.err('登录失败', e.status === 401 ? '密码错误' : e.message)
    password.value = ''
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.login-wrap {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 28px 18px;
}
.login-card {
  width: 100%;
  max-width: 380px;
  padding: 34px 32px 30px;
}
.pw-wrap {
  position: relative;
}
.pw-eye {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: 0;
  color: var(--ink-faint);
  cursor: pointer;
  padding: 4px;
  display: grid;
  place-items: center;
}
.pw-eye:hover {
  color: var(--ink);
}
</style>
