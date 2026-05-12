<template>
  <div>
    <Dialog
      :show="loginStore.showLogin"
      :buttons="dialogConfig.buttons"
      width="1000px"
      :showCancel="false"
      @close="closeDialog"
      :padding="0"
      :draggable="false"
      :top="100"
      dialogClass="login-dialog"
    >
      <div class="dialog-panel">
        <div class="bg">
          <img :src="proxy.Utils.getLocalImage('login_bg.png')" />
        </div>
        <el-form
          class="login-register"
          :model="formData"
          :rules="rules"
          ref="formDataRef"
        >
          <div class="tab-panel">
            <div :class="[opType == 0 ? '' : 'active']" @click="showPanel(1)">
              登录
            </div>
            <el-divider direction="vertical" />
            <div :class="[opType == 1 ? '' : 'active']" @click="showPanel(0)">
              注册
            </div>
          </div>
          <!--input输入-->
          <el-form-item prop="email">
            <el-input
              size="large"
              clearable
              placeholder="请输入邮箱"
              v-model="formData.email"
              maxLength="150"
            >
              <template #prefix>
                <span class="iconfont icon-account"></span>
              </template>
            </el-input>
          </el-form-item>
          <!--登录密码-->
          <el-form-item prop="password" v-if="opType == 1">
            <el-input
              show-password
              size="large"
              placeholder="请输入密码"
              v-model="formData.password"
            >
              <template #prefix>
                <span class="iconfont icon-password"></span>
              </template>
            </el-input>
          </el-form-item>
          <!--注册-->
          <div v-if="opType == 0">
            <el-form-item prop="nickName" v-if="opType == 0">
              <el-input
                size="large"
                clearable
                placeholder="请输入昵称"
                v-model="formData.nickName"
                maxLength="20"
              >
                <template #prefix>
                  <span class="iconfont icon-account"></span>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item prop="registerPassword">
              <el-input
                show-password
                type="password"
                size="large"
                placeholder="请输入密码"
                v-model="formData.registerPassword"
              >
                <template #prefix>
                  <span class="iconfont icon-password"></span>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item prop="reRegisterPassword">
              <el-input
                show-password
                type="password"
                size="large"
                placeholder="请再次输入密码"
                v-model="formData.reRegisterPassword"
              >
                <template #prefix>
                  <span class="iconfont icon-password"></span>
                </template>
              </el-input>
            </el-form-item>
          </div>
          <el-form-item prop="checkCode">
            <div class="check-code-panel">
              <el-input
                size="large"
                placeholder="请输入验证码"
                v-model="formData.checkCode"
                @keyup.enter="doSubmit"
              >
                <template #prefix>
                  <span class="iconfont icon-checkcode"></span>
                </template>
              </el-input>
              <img
                :src="checkCodeInfo.checkCode"
                class="check-code"
                @click="changeCheckCode()"
              />
            </div>
          </el-form-item>
          <el-form-item class="bottom-btn">
            <el-button
              type="primary"
              size="large"
              class="login-btn"
              @click="doSubmit"
            >
              <span v-if="opType == 0">注册</span>
              <span v-if="opType == 1">登录</span>
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </Dialog>
  </div>
</template>

<script setup>
import {
  ref,
  reactive,
  getCurrentInstance,
  nextTick,
  onMounted,
  onUpdated,
} from "vue";
import { useRouter, useRoute } from "vue-router";
import md5 from "js-md5";
const { proxy } = getCurrentInstance();
const router = useRouter();
const route = useRoute();

import { useLoginStore } from "@/stores/loginStore.js";
const loginStore = useLoginStore();

//验证码
const checkCodeInfo = ref({});
const changeCheckCode = async () => {
  let result = await proxy.Request({
    url: proxy.Api.checkCode,
  });
  if (!result) {
    return;
  }
  checkCodeInfo.value = result.data;
};

//登录，注册 弹出配置
const dialogConfig = ref({
  show: true,
});

const checkRePassword = (rule, value, callback) => {
  if (value !== formData.value.registerPassword) {
    callback(new Error(rule.message));
  } else {
    callback();
  }
};

// 0:注册 1:登录
const opType = ref(1);
const formData = ref({});
const formDataRef = ref();
const rules = {
  email: [
    { required: true, message: "请输入邮箱" },
    { validator: proxy.Verify.email, message: "请输入正确的邮箱" },
  ],
  password: [{ required: true, message: "请输入密码" }],
  nickName: [{ required: true, message: "请输入昵称" }],
  registerPassword: [
    { required: true, message: "请输入密码" },
    {
      validator: proxy.Verify.password,
      message: "密码只能是数字，字母，特殊字符 8-18位",
    },
  ],
  reRegisterPassword: [
    { required: true, message: "请再次输入密码" },
    {
      validator: checkRePassword,
      message: "两次输入的密码不一致",
    },
  ],
  checkCode: [{ required: true, message: "请输入图片验证码" }],
};

//重置表单
const resetForm = () => {
  changeCheckCode();
  nextTick(() => {
    formDataRef.value.resetFields();
    formData.value = {};
  });
};

// 登录、注册、重置密码  提交表单
const doSubmit = () => {
  formDataRef.value.validate(async (valid) => {
    if (!valid) {
      return;
    }
    let params = {};
    Object.assign(params, formData.value);
    params.checkCodeKey = checkCodeInfo.value.checkCodeKey;
    //登录
    if (opType.value == 1) {
      params.password = md5(params.password);
    }
    let result = await proxy.Request({
      url: opType.value == 0 ? proxy.Api.register : proxy.Api.login,
      params: params,
      errorCallback: () => {
        changeCheckCode();
      },
    });
    if (!result) {
      return;
    }
    //注册返回
    if (opType.value == 0) {
      proxy.Message.success("注册成功,请登录");
      showPanel(1);
    } else if (opType.value == 1) {
      proxy.Message.success("登录成功");
      loginStore.setLogin(false);
      loginStore.saveUserInfo(result.data);
    }
  });
};

const closeDialog = () => {
  dialogConfig.value.show = false;
  loginStore.setLogin(false);
};

const showPanel = (type) => {
  opType.value = type;
  if (loginStore.showLogin) {
    resetForm();
  }
};

onUpdated(() => {
  showPanel(1);
});

onMounted(() => {
  showPanel(1);
});
</script>

<style lang="scss">
.cust-dialog.login-dialog {
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 24px 70px rgba(60, 72, 100, 0.22);
  animation: loginDialogIn 0.38s ease-out both;

  .el-dialog__header {
    position: absolute;
    top: 0;
    right: 0;
    z-index: 6;
    padding: 16px;
  }

  .el-dialog__body {
    padding: 0;
  }

  .dialog-body {
    overflow: hidden;
  }
}

.login-dialog {
  .dialog-panel {
    position: relative;
    display: grid;
    grid-template-columns: 500px 1fr;
    align-items: center;
    min-height: 580px;
    overflow: hidden;
    background:
      radial-gradient(circle at 76% 18%, rgba(255, 123, 179, 0.18), transparent 25%),
      radial-gradient(circle at 82% 78%, rgba(97, 191, 255, 0.16), transparent 24%),
      linear-gradient(120deg, #fff 0%, #fff 52%, #fbfdff 100%);

    &::before,
    &::after {
      content: "";
      position: absolute;
      pointer-events: none;
      z-index: 0;
    }

    &::before {
      right: 70px;
      top: 78px;
      width: 170px;
      height: 170px;
      border-radius: 50%;
      background: radial-gradient(circle, rgba(255, 126, 178, 0.26), transparent 68%);
      filter: blur(2px);
      animation: formGlowFloat 7s ease-in-out infinite;
    }

    &::after {
      right: 40px;
      bottom: 55px;
      width: 240px;
      height: 240px;
      border-radius: 50%;
      background: radial-gradient(circle, rgba(90, 188, 255, 0.16), transparent 70%);
      animation: formGlowFloat 8s ease-in-out -2s infinite;
    }
  }

  .bg {
    position: relative;
    z-index: 1;
    width: 100%;
    height: 100%;
    min-height: 580px;
    overflow: hidden;
    flex-shrink: 0;
    background: #eef9f6;

    &::before,
    &::after {
      content: "";
      position: absolute;
      inset: 0;
      pointer-events: none;
      z-index: 2;
    }

    &::before {
      opacity: 0.76;
      background:
        radial-gradient(circle at 18% 22%, rgba(255, 255, 255, 0.72) 0 3px, transparent 4px),
        radial-gradient(circle at 80% 26%, rgba(255, 136, 188, 0.28) 0 8px, transparent 9px),
        radial-gradient(circle at 26% 78%, rgba(96, 190, 255, 0.24) 0 10px, transparent 11px),
        radial-gradient(circle at 74% 72%, rgba(255, 222, 114, 0.25) 0 7px, transparent 8px);
      background-size: 130px 150px, 190px 170px, 230px 210px, 160px 180px;
      animation: animeLightMove 14s linear infinite;
    }

    &::after {
      background:
        linear-gradient(90deg, rgba(255, 255, 255, 0) 62%, rgba(255, 255, 255, 0.28) 100%),
        linear-gradient(180deg, rgba(255, 255, 255, 0.08), rgba(76, 150, 180, 0.12));
      mix-blend-mode: screen;
    }

    img {
      width: 100%;
      height: 100%;
      display: block;
      object-fit: cover;
      object-position: center;
      transform-origin: center;
      animation: animeImageFloat 8s ease-in-out infinite;
      will-change: transform;
    }
  }

  .login-register {
    position: relative;
    z-index: 2;
    width: 350px;
    justify-self: center;
    padding: 34px 0;
    animation: formSlideIn 0.5s ease-out 0.08s both;

    &::before {
      content: "";
      position: absolute;
      inset: -34px -42px;
      z-index: -1;
      border-radius: 24px;
      background:
        linear-gradient(145deg, rgba(255, 255, 255, 0.78), rgba(255, 255, 255, 0.36)),
        radial-gradient(circle at 12% 12%, rgba(255, 135, 188, 0.14), transparent 30%),
        radial-gradient(circle at 90% 85%, rgba(80, 183, 255, 0.14), transparent 34%);
      filter: drop-shadow(0 18px 35px rgba(64, 158, 255, 0.08));
    }

    .tab-panel {
      margin: 10px auto 22px;
      display: flex;
      width: 140px;
      font-size: 18px;
      align-items: center;
      justify-content: space-around;
      cursor: pointer;

      .active {
        position: relative;
        color: var(--blue2);
        font-weight: 600;

        &::after {
          content: "";
          position: absolute;
          left: 50%;
          bottom: -8px;
          width: 22px;
          height: 3px;
          border-radius: 999px;
          background: linear-gradient(90deg, #67b8ff, #ff78b1);
          transform: translateX(-50%);
        }
      }
    }

    .el-form-item {
      animation: formItemIn 0.42s ease-out both;
    }

    .el-input__wrapper {
      border-radius: 8px;
      transition:
        box-shadow 0.2s ease,
        transform 0.2s ease,
        border-color 0.2s ease;
    }

    .el-input__wrapper:hover,
    .el-input__wrapper.is-focus {
      transform: translateY(-1px);
      box-shadow:
        0 0 0 1px rgba(97, 179, 255, 0.26) inset,
        0 8px 22px rgba(79, 163, 255, 0.13);
    }

    .iconfont {
      color: #9aa7b8;
      transition: color 0.2s ease;
    }

    .el-input__wrapper:hover .iconfont,
    .el-input__wrapper.is-focus .iconfont {
      color: #5aaeff;
    }

    .no-account {
      width: 100%;
      display: flex;
      justify-content: space-between;
    }

    .login-btn {
      position: relative;
      width: 100%;
      overflow: hidden;
      border: 0;
      color: #fff;
      background: linear-gradient(90deg, #58aaff 0%, #7b9dff 48%, #ff7aae 100%);
      box-shadow: 0 10px 24px rgba(82, 154, 255, 0.24);
      transition:
        transform 0.2s ease,
        box-shadow 0.2s ease,
        filter 0.2s ease;

      &::before {
        content: "";
        position: absolute;
        top: 0;
        left: -70%;
        width: 46%;
        height: 100%;
        background: linear-gradient(120deg, transparent, rgba(255, 255, 255, 0.55), transparent);
        transform: skewX(-24deg);
      }

      &:hover {
        transform: translateY(-2px);
        filter: saturate(1.06);
        box-shadow: 0 14px 30px rgba(82, 154, 255, 0.32);

        &::before {
          animation: buttonShine 0.78s ease;
        }
      }
    }

    .bottom-btn {
      margin-bottom: 0px;
    }
  }

  .check-code-panel {
    display: flex;
    width: 100%;

    .el-input {
      flex: 1;
    }

    .check-code {
      width: 122px;
      height: 40px;
      margin-left: 8px;
      cursor: pointer;
      object-fit: cover;
      border-radius: 6px;
      transition:
        transform 0.2s ease,
        box-shadow 0.2s ease;

      &:hover {
        transform: translateY(-1px) scale(1.03) rotate(-1deg);
        box-shadow: 0 8px 18px rgba(80, 160, 255, 0.18);
      }
    }
  }
}

@keyframes loginDialogIn {
  from {
    opacity: 0;
    transform: translateY(18px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@keyframes animeImageFloat {
  0%,
  100% {
    transform: scale(1.04) translate3d(0, 0, 0);
  }
  50% {
    transform: scale(1.08) translate3d(-8px, -6px, 0);
  }
}

@keyframes animeLightMove {
  from {
    background-position: 0 0, 0 0, 0 0, 0 0;
  }
  to {
    background-position: 130px 150px, -190px 170px, 230px -210px, -160px -180px;
  }
}

@keyframes formSlideIn {
  from {
    opacity: 0;
    transform: translateX(24px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes formItemIn {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes formGlowFloat {
  0%,
  100% {
    transform: translate3d(0, 0, 0) scale(1);
  }
  50% {
    transform: translate3d(10px, -8px, 0) scale(1.06);
  }
}

@keyframes buttonShine {
  from {
    left: -70%;
  }
  to {
    left: 120%;
  }
}

@media (max-width: 900px) {
  .cust-dialog.login-dialog {
    width: calc(100vw - 32px) !important;
  }

  .login-dialog {
    .dialog-panel {
      grid-template-columns: 1fr;
      min-height: auto;
    }

    .bg {
      display: none;
    }

    .login-register {
      width: min(350px, calc(100vw - 80px));
      padding: 54px 0 42px;
    }
  }
}

@media (prefers-reduced-motion: reduce) {
  .cust-dialog.login-dialog,
  .login-dialog .bg::before,
  .login-dialog .bg img,
  .login-dialog .dialog-panel::before,
  .login-dialog .dialog-panel::after,
  .login-dialog .login-register,
  .login-dialog .login-register .el-form-item {
    animation: none;
  }
}
</style>
