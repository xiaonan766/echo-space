<template>
  <div class="setting-form">
    <el-form
      :model="formData"
      :rules="rules"
      ref="formDataRef"
      label-width="180px"
      @submit.prevent
    >
      <div class="setting-section-title">基础设置</div>
      <el-form-item label="注册赠送硬币数量" prop="registerCoinCount">
        <el-input
          placeholder="请输入注册赠送硬币数量"
          v-model="formData.registerCoinCount"
          type="number"
          :min="1"
        />
      </el-form-item>
      <el-form-item label="发布视频赠送硬币数量" prop="postVideoCoinCount">
        <el-input
          placeholder="请输入发布视频赠送硬币数量"
          v-model="formData.postVideoCoinCount"
          type="number"
          :min="1"
        />
      </el-form-item>
      <el-form-item label="单个视频大小" prop="videoSize">
        <el-input
          placeholder="请输入单个视频大小"
          v-model="formData.videoSize"
          type="number"
          :min="1"
        >
          <template #suffix>MB</template>
        </el-input>
      </el-form-item>
      <el-form-item label="稿件最大分P数量" prop="videoPCount">
        <el-input
          placeholder="请输入稿件最大分P数量"
          v-model="formData.videoPCount"
          type="number"
          :min="1"
        />
      </el-form-item>
      <el-form-item label="每天允许发布视频数" prop="videoCount">
        <el-input
          placeholder="请输入每天允许发布视频数"
          v-model="formData.videoCount"
          type="number"
          :min="1"
        />
      </el-form-item>
      <el-form-item label="每天允许评论数" prop="commentCount">
        <el-input
          placeholder="请输入每天允许评论数"
          v-model="formData.commentCount"
          type="number"
          :min="1"
        />
      </el-form-item>
      <el-form-item label="每天允许弹幕数" prop="danmuCount">
        <el-input
          placeholder="请输入每天允许弹幕数"
          v-model="formData.danmuCount"
          type="number"
          :min="1"
        />
      </el-form-item>

      <div class="setting-section-title">弹幕限流设置</div>
      <el-form-item label="用户限流条数" prop="danmuUserRateCount">
        <el-input
          placeholder="请输入用户限流条数"
          v-model="formData.danmuUserRateCount"
          type="number"
          :min="1"
        />
      </el-form-item>
      <el-form-item label="用户限流秒数" prop="danmuUserRateSeconds">
        <el-input
          placeholder="请输入用户限流秒数"
          v-model="formData.danmuUserRateSeconds"
          type="number"
          :min="1"
        />
      </el-form-item>
      <el-form-item label="用户视频限流条数" prop="danmuUserVideoRateCount">
        <el-input
          placeholder="请输入用户视频限流条数"
          v-model="formData.danmuUserVideoRateCount"
          type="number"
          :min="1"
        />
      </el-form-item>
      <el-form-item label="用户视频限流秒数" prop="danmuUserVideoRateSeconds">
        <el-input
          placeholder="请输入用户视频限流秒数"
          v-model="formData.danmuUserVideoRateSeconds"
          type="number"
          :min="1"
        />
      </el-form-item>
      <el-form-item label="IP 限流条数" prop="danmuIPRateCount">
        <el-input
          placeholder="请输入 IP 限流条数"
          v-model="formData.danmuIPRateCount"
          type="number"
          :min="1"
        />
      </el-form-item>
      <el-form-item label="IP 限流秒数" prop="danmuIPRateSeconds">
        <el-input
          placeholder="请输入 IP 限流秒数"
          v-model="formData.danmuIPRateSeconds"
          type="number"
          :min="1"
        />
      </el-form-item>
      <el-form-item label="视频限流条数" prop="danmuVideoRateCount">
        <el-input
          placeholder="请输入视频限流条数"
          v-model="formData.danmuVideoRateCount"
          type="number"
          :min="1"
        />
      </el-form-item>
      <el-form-item label="视频限流秒数" prop="danmuVideoRateSeconds">
        <el-input
          placeholder="请输入视频限流秒数"
          v-model="formData.danmuVideoRateSeconds"
          type="number"
          :min="1"
        />
      </el-form-item>

      <el-form-item>
        <el-button type="primary" @click="saveSetting">保存</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { ref, getCurrentInstance } from "vue";

const { proxy } = getCurrentInstance();

const defaultFormData = {
  registerCoinCount: 10,
  postVideoCoinCount: 5,
  videoSize: 150,
  videoPCount: 10,
  videoCount: 10,
  commentCount: 20,
  danmuCount: 20,
  danmuUserRateCount: 1,
  danmuUserRateSeconds: 2,
  danmuUserVideoRateCount: 5,
  danmuUserVideoRateSeconds: 60,
  danmuIPRateCount: 30,
  danmuIPRateSeconds: 60,
  danmuVideoRateCount: 300,
  danmuVideoRateSeconds: 60,
};

const formData = ref({ ...defaultFormData });
const formDataRef = ref();

const requiredNumberRule = (message) => [
  { required: true, message, trigger: "blur" },
];

const rules = {
  registerCoinCount: requiredNumberRule("请输入注册赠送硬币数量"),
  postVideoCoinCount: requiredNumberRule("请输入发布视频赠送硬币数量"),
  videoSize: requiredNumberRule("请输入单个视频大小"),
  videoPCount: requiredNumberRule("请输入稿件最大分P数量"),
  videoCount: requiredNumberRule("请输入每天允许发布视频数"),
  commentCount: requiredNumberRule("请输入每天允许评论数"),
  danmuCount: requiredNumberRule("请输入每天允许弹幕数"),
  danmuUserRateCount: requiredNumberRule("请输入用户限流条数"),
  danmuUserRateSeconds: requiredNumberRule("请输入用户限流秒数"),
  danmuUserVideoRateCount: requiredNumberRule("请输入用户视频限流条数"),
  danmuUserVideoRateSeconds: requiredNumberRule("请输入用户视频限流秒数"),
  danmuIPRateCount: requiredNumberRule("请输入 IP 限流条数"),
  danmuIPRateSeconds: requiredNumberRule("请输入 IP 限流秒数"),
  danmuVideoRateCount: requiredNumberRule("请输入视频限流条数"),
  danmuVideoRateSeconds: requiredNumberRule("请输入视频限流秒数"),
};

const getSetting = async () => {
  const result = await proxy.Request({
    url: proxy.Api.getSetting,
  });
  if (!result) {
    return;
  }
  formData.value = {
    ...defaultFormData,
    ...(result.data || {}),
  };
};
getSetting();

const saveSetting = () => {
  formDataRef.value.validate(async (valid) => {
    if (!valid) {
      return;
    }
    const params = {};
    Object.keys(defaultFormData).forEach((key) => {
      params[key] = Number(formData.value[key]);
    });

    const result = await proxy.Request({
      url: proxy.Api.saveSetting,
      params,
    });
    if (!result) {
      return;
    }
    proxy.Message.success("保存成功");
  });
};
</script>

<style lang="scss" scoped>
.setting-form {
  padding: 20px;
  width: 680px;

  .setting-section-title {
    margin: 4px 0 18px;
    padding-left: 8px;
    border-left: 3px solid #409eff;
    color: #333;
    font-size: 15px;
    font-weight: 600;
  }

  :deep(.el-input) {
    width: 320px;
  }
}
</style>
