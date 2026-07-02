<template>
  <div class="gallery-image-item" :style="{ 'margin-top': marginTop + 'px' }">
    <router-link :to="`/gallery/${data.imageId}`" target="_blank">
      <div class="cover">
        <Cover :source="data.imageCover"></Cover>
        <div class="shade">
          <div class="image-type">图片</div>
        </div>
      </div>
    </router-link>
    <div class="image-info">
      <router-link
        class="title"
        :to="`/gallery/${data.imageId}`"
        target="_blank"
      >
        {{ data.imageName }}
      </router-link>
      <router-link
        class="user-name"
        :to="`/user/${data.userId}`"
        target="_blank"
      >
        <span class="iconfont icon-upzhu">{{ data.nickName }} · </span>
        <span>{{ proxy.Utils.formatDate(data.createTime) }}</span>
      </router-link>
    </div>
  </div>
</template>

<script setup>
import { getCurrentInstance } from 'vue'

const { proxy } = getCurrentInstance()

defineProps({
  data: {
    type: Object,
    default: () => ({}),
  },
  marginTop: {
    type: Number,
    default: 0,
  },
})
</script>

<style lang="scss" scoped>
.gallery-image-item {
  width: 100%;
  overflow: hidden;
  .cover {
    cursor: pointer;
    position: relative;
    overflow: hidden;
    border-radius: 5px;
    .shade {
      position: absolute;
      bottom: 0;
      left: 0;
      z-index: 1;
      box-sizing: border-box;
      padding: 8px 8px 6px;
      width: 100%;
      height: 38px;
      border-bottom-right-radius: 6px;
      border-bottom-left-radius: 6px;
      background-image: linear-gradient(
        180deg,
        rgba(0, 0, 0, 0) 0%,
        rgba(0, 0, 0, 0.8) 100%
      );
      color: #fff;
      display: flex;
      align-items: center;
      justify-content: flex-end;
      .image-type {
        font-size: 13px;
      }
    }
  }
  .image-info {
    cursor: pointer;
    .title {
      height: 40px;
      color: var(--text2);
      font-size: 14px;
      margin-top: 10px;
      display: -webkit-box;
      overflow: hidden;
      text-decoration: none;
      -webkit-box-orient: vertical;
      text-overflow: ellipsis;
      word-break: break-word !important;
      line-break: anywhere;
      -webkit-line-clamp: 2;
      cursor: pointer;
      &:hover {
        color: var(--blue);
      }
    }
    .user-name {
      margin-top: 5px;
      color: #9499a0;
      font-size: 13px;
      cursor: pointer;
      text-decoration: none;
      display: block;
      &:hover {
        color: var(--blue);
      }
      .iconfont {
        &::before {
          font-size: 18px;
          margin-right: 3px;
          float: left;
        }
        font-size: 13px;
      }
    }
  }
}
</style>
