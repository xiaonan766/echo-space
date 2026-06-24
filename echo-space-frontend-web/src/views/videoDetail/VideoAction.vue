<template>
  <div class="action-panel">
    <div
      :class="[
        'iconfont icon-like-solid',
        videoInfo.likeCountActive ? 'active' : '',
      ]"
      @click="userAction('VIDEO_LIKE')"
    >
      {{ videoInfo.likeCount }}
    </div>
    <div
      :class="[
        'iconfont icon-toubi',
        videoInfo.coinCountActive ? 'active' : '',
      ]"
      @click="userActionCoin('VIDEO_COIN')"
    >
      {{ videoInfo.coinCount }}
    </div>
    <div
      :class="[
        'iconfont icon-collection-solid',
        videoInfo.collectCountActive ? 'active' : '',
      ]"
      @click="userAction('VIDEO_COLLECT')"
    >
      {{ videoInfo.collectCount }}
    </div>
    <div
      v-if="showDownloadButton"
      class="iconfont icon-down download-action"
      @click="downloadVideo"
    >
      下载
    </div>
  </div>
  <VideoCoin ref="videoCoinRef"></VideoCoin>
</template>

<script setup>
import VideoCoin from "./VideoCoin.vue";
import { doUserAction } from "@/utils/Api";
import { ACTION_TYPE } from "@/utils/Constants";

import { useLoginStore } from "@/stores/loginStore";
const loginStore = useLoginStore();

import { computed, ref, reactive, getCurrentInstance, nextTick, inject } from "vue";
const { proxy } = getCurrentInstance();
import { useRoute, useRouter } from "vue-router";
const route = useRoute();
const router = useRouter();

const videoInfo = inject("videoInfo");
const currentVideoFile = inject("currentVideoFile");

const userAction = (type) => {
  if (Object.keys(loginStore.userInfo).length == 0) {
    loginStore.setLogin(true);
    return;
  }
  doUserAction(
    {
      videoId: route.params.videoId,
      actionType: ACTION_TYPE[type].value,
    },
    () => {
      if (type == "VIDEO_LIKE") {
        if (videoInfo.value.likeCountActive) {
          videoInfo.value.likeCountActive = false;
          videoInfo.value.likeCount--;
        } else {
          videoInfo.value.likeCountActive = true;
          videoInfo.value.likeCount++;
        }
      } else if (type == "VIDEO_COLLECT") {
        if (videoInfo.value.collectCountActive) {
          videoInfo.value.collectCountActive = false;
          videoInfo.value.collectCount--;
        } else {
          videoInfo.value.collectCountActive = true;
          videoInfo.value.collectCount++;
        }
      }
    }
  );
};

const videoCoinRef = ref();
const userActionCoin = () => {
  if (Object.keys(loginStore.userInfo).length == 0) {
    loginStore.setLogin(true);
    return;
  }
  if (videoInfo.value.coinCountActive) {
    proxy.Message.warning("对本稿件的投币枚数已用完");
    return;
  }
  videoCoinRef.value.show();
};

const downloadVideo = () => {
  const fileId = currentVideoFile.value?.fileId;
  if (!fileId) {
    proxy.Message.warning("当前视频文件不存在");
    return;
  }
  if (Number(currentVideoFile.value?.downloadStatus) !== 2) {
    proxy.Message.warning("视频下载文件正在生成，请稍后再试");
    return;
  }
  const downloadUrl = `/api${proxy.Api.downloadVideo}/${encodeURIComponent(fileId)}`;
  window.open(downloadUrl, "_blank");
};

const showDownloadButton = computed(() => {
  const downloadPermission = videoInfo.value.downloadPermission;
  const canDownload =
    downloadPermission === undefined ||
    downloadPermission === null ||
    Number(downloadPermission) === 1;
  return canDownload && !!currentVideoFile.value?.fileId;
});
</script>

<style lang="scss" scoped>
.action-panel {
  display: flex;
  align-items: center;
  border-bottom: 1px solid #e3e5e7;
  padding: 20px 0px;
  .iconfont {
    cursor: pointer;
    color: #61666d;
    display: flex;
    align-items: center;
    margin-right: 40px;
    &::before {
      margin-right: 10px;
      font-size: 35px;
    }

    &:hover {
      color: #4d4e4f;
    }
  }
  .active {
    &::before {
      color: var(--blue);
    }
  }
  .download-action {
    &::before {
      color: #61666d;
    }
  }
}
</style>
