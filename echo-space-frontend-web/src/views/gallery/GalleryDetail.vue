<template>
  <div class="gallery-detail" v-if="Object.keys(imageInfo).length > 0">
    <div class="detail-header">
      <div class="title-panel">
        <div class="title">{{ imageInfo.imageName }}</div>
        <div class="meta">
          <span>{{ imageInfo.createTime }}</span>
          <span>共 {{ imageList.length }} 张图片</span>
        </div>
      </div>
      <div class="author-panel">
        <Avatar
          :userId="imageInfo.userId"
          :avatar="imageInfo.avatar"
          :width="48"
        ></Avatar>
        <div class="author-info">
          <router-link
            class="nick-name"
            :to="`/user/${imageInfo.userId}`"
            target="_blank"
          >
            {{ imageInfo.nickName }}
          </router-link>
          <router-link
            class="btn-go-home"
            :to="`/user/${imageInfo.userId}`"
            target="_blank"
          >
            访问主页
          </router-link>
        </div>
      </div>
    </div>

    <div class="image-list">
      <div class="image-view" v-for="(item, index) in imageList" :key="item.fileId">
        <div class="image-index">图 {{ index + 1 }}</div>
        <el-image
          :src="`${proxy.Api.sourcePath}${item.sourceName}`"
          :preview-src-list="previewList"
          :initial-index="index"
          fit="contain"
          preview-teleported
        ></el-image>
      </div>
    </div>

    <div class="summary-panel">
      <div
        class="summary"
        v-if="imageInfo.introduction"
        v-html="imageInfo.introduction"
      ></div>
      <div class="summary empty" v-else>这个图片稿件还没有简介</div>
      <div class="tag-list" v-if="imageInfo.tags && imageInfo.tags.length > 0">
        <router-link
          :to="`/gallery?mode=text&keyword=${encodeURIComponent(item)}`"
          class="tag-item"
          target="_blank"
          v-for="item in imageInfo.tags"
          :key="item"
        >
          {{ item }}
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, getCurrentInstance, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useNavAction } from '@/stores/navActionStore'

const { proxy } = getCurrentInstance()
const route = useRoute()
const navActionStore = useNavAction()

const imageInfo = ref({})
const imageList = ref([])

const previewList = computed(() => {
  return imageList.value.map((item) => `${proxy.Api.sourcePath}${item.sourceName}`)
})

const loadImageInfo = async () => {
  const result = await proxy.Request({
    url: proxy.Api.getGalleryImageInfo,
    params: {
      imageId: route.params.imageId,
    },
  })
  if (!result) {
    return
  }
  const resultImageInfo = result.data.imageInfo || {}
  resultImageInfo.introduction = proxy.Utils.resetHtmlContent(
    resultImageInfo.introduction || ''
  )
  resultImageInfo.tags = resultImageInfo.tags
    ? resultImageInfo.tags.split(',').filter((item) => item)
    : []
  imageInfo.value = resultImageInfo
  imageList.value = result.data.imageList || []
}

onMounted(() => {
  navActionStore.setShowHeader(false)
  navActionStore.setFixedHeader(true)
  navActionStore.setFixedCategory(false)
  navActionStore.setShowCategory(false)
  navActionStore.setForceFixedHeader(true)
  loadImageInfo()
})
</script>

<style lang="scss" scoped>
.gallery-detail {
  margin-top: 64px;
  min-height: calc(100vh - 64px);
  padding: 24px 0 40px;
  .detail-header {
    display: flex;
    align-items: flex-start;
    border-bottom: 1px solid #e3e5e7;
    padding-bottom: 20px;
    .title-panel {
      flex: 1;
      min-width: 0;
      .title {
        color: var(--text);
        font-size: 22px;
        line-height: 32px;
        word-break: break-word;
      }
      .meta {
        margin-top: 8px;
        color: #9499a0;
        font-size: 14px;
        span {
          margin-right: 18px;
        }
      }
    }
    .author-panel {
      width: 300px;
      margin-left: 30px;
      display: flex;
      align-items: center;
      .author-info {
        margin-left: 12px;
        min-width: 0;
        flex: 1;
        .nick-name {
          color: var(--text);
          font-size: 16px;
          text-decoration: none;
          display: block;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
          &:hover {
            color: var(--blue);
          }
        }
        .btn-go-home {
          display: inline-block;
          margin-top: 8px;
          text-decoration: none;
          color: #fb7299;
          border: 1px solid #fb7299;
          line-height: 28px;
          border-radius: 5px;
          padding: 0 18px;
          &:hover {
            background: #ffecf1;
          }
        }
      }
    }
  }
  .image-list {
    margin-top: 24px;
    .image-view {
      background: #f6f7f8;
      border-radius: 8px;
      margin-bottom: 20px;
      padding: 14px;
      .image-index {
        color: #9499a0;
        font-size: 13px;
        margin-bottom: 10px;
      }
      :deep(.el-image) {
        width: 100%;
        display: block;
        text-align: center;
      }
      :deep(img) {
        max-height: 78vh;
      }
    }
  }
  .summary-panel {
    border-top: 1px solid #e3e5e7;
    padding: 20px 0 0;
    .summary {
      color: var(--text2);
      line-height: 24px;
      word-break: break-word;
    }
    .empty {
      color: #9499a0;
    }
    .tag-list {
      margin-top: 20px;
      display: flex;
      flex-wrap: wrap;
      .tag-item {
        cursor: pointer;
        text-decoration: none;
        color: var(--text2);
        background: #f1f2f3;
        border-radius: 16px;
        height: 32px;
        line-height: 32px;
        padding: 0 12px;
        margin: 0 12px 8px 0;
        &:hover {
          color: var(--blue);
        }
      }
    }
  }
}
</style>
