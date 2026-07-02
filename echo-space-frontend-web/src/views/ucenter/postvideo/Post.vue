<template>
  <div class="upload-video-panel">
    <div class="post-type-tabs" v-if="!videoId">
      <div
        :class="['tab-item', contentType === CONTENT_TYPE_VIDEO ? 'active' : '']"
        @click="handleSwitchContentType(CONTENT_TYPE_VIDEO)"
      >
        视频投稿
      </div>
      <div
        :class="['tab-item', contentType === CONTENT_TYPE_IMAGE ? 'active' : '']"
        @click="handleSwitchContentType(CONTENT_TYPE_IMAGE)"
      >
        图片投稿
      </div>
    </div>

    <div v-show="contentType === CONTENT_TYPE_VIDEO">
      <VideoUploader ref="videoUploaderRef"></VideoUploader>
    </div>
    <div v-show="contentType === CONTENT_TYPE_IMAGE">
      <ImagePostUploader
        ref="imagePostUploaderRef"
        @change="handleImageListChange"
      ></ImagePostUploader>
    </div>

    <div v-if="showPostForm" class="video-form">
      <el-form :model="formData" :rules="rules" ref="formDataRef" label-width="70px" @submit.prevent>
        <el-form-item label="封面" prop="videoCover" v-if="contentType === CONTENT_TYPE_VIDEO">
          <ImageCoverSelect :coverWidth="200" :cutWidth="680" :scale="0.6" :coverImage="formData.videoCover">
          </ImageCoverSelect>
        </el-form-item>
        <el-form-item label="封面" prop="videoCover" v-else>
          <ImageCoverSelect :coverWidth="200" :cutWidth="640" :scale="0.5625" :coverImage="formData.videoCover">
          </ImageCoverSelect>
        </el-form-item>
        <el-form-item label="标题" prop="videoName">
          <el-input clearable placeholder="请输入标题" v-model="formData.videoName" maxlength="100"
            show-word-limit></el-input>
        </el-form-item>
        <el-form-item label="类型" prop="postType">
          <el-radio-group v-model="formData.postType">
            <el-radio :value="0">自制</el-radio>
            <el-radio :value="1">转载</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="" prop="originInfo" v-if="formData.postType == 1">
          <el-input clearable :placeholder="originPlaceholder"
            v-model="formData.originInfo" maxlength="200" show-word-limit></el-input>
        </el-form-item>
        <el-form-item label="标签" prop="tags">
          <TagInput v-model="formData.tags"></TagInput>
        </el-form-item>
        <el-form-item label="分区" prop="categoryArray">
          <el-cascader v-model="formData.categoryArray" :options="categoryStore.categoryList"
            :props="{ value: 'categoryId', label: 'categoryName' }" />
        </el-form-item>
        <el-form-item label="简介" prop="introduction">
          <el-input clearable :placeholder="introductionPlaceholder" type="textarea" :rows="5" :maxlength="2000"
            resize="none" show-word-limit v-model="formData.introduction"></el-input>
        </el-form-item>
        <template v-if="contentType === CONTENT_TYPE_VIDEO">
          <el-form-item label="互动设置" prop="introduction">
            <el-checkbox-group v-model="formData.interactionArray">
              <el-checkbox value="0">关闭弹幕</el-checkbox>
              <el-checkbox value="1">关闭评论</el-checkbox>
            </el-checkbox-group>
          </el-form-item>
          <el-form-item class="download-setting-item" label="下载设置" prop="downloadPermission">
            <el-radio-group v-model="formData.downloadPermission">
              <el-radio :value="1">允许下载</el-radio>
              <el-radio :value="0">禁止下载</el-radio>
            </el-radio-group>
          </el-form-item>
        </template>
        <el-form-item label="">
          <el-button type="primary" @click="submitForm">立即投稿</el-button>
          <el-button @click="router.push('/ucenter/video')">取消</el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { useCategoryStore } from '@/stores/categoryStore.js'
const categoryStore = useCategoryStore()
import TagInput from './TagInput.vue'
import VideoUploader from './VideoUploader.vue'
import ImagePostUploader from './ImagePostUploader.vue'
import {
  computed,
  getCurrentInstance,
  nextTick,
  onMounted,
  onUnmounted,
  ref,
  watch,
  provide,
} from 'vue'

import { uploadImage } from '@/utils/Api.js'

const { proxy } = getCurrentInstance()
import { useRoute, useRouter } from 'vue-router'
const route = useRoute()
const router = useRouter()

import { mitter } from '@/eventbus/eventBus.js'

const CONTENT_TYPE_VIDEO = 0
const CONTENT_TYPE_IMAGE = 1

const contentType = ref(CONTENT_TYPE_VIDEO)
const startUpload = ref(false)
const startImagePost = ref(false)
const imagePostList = ref([])

const createDefaultFormData = (fileName = '') => ({
  tags: [],
  videoName: fileName,
  postType: 0,
  downloadPermission: 1,
  contentType: contentType.value,
})

const formData = ref(createDefaultFormData())
const formDataRef = ref()
const videoId = ref()

const showPostForm = computed(() => {
  if (contentType.value === CONTENT_TYPE_VIDEO) {
    return startUpload.value
  }
  return startImagePost.value
})

const originPlaceholder = computed(() => {
  return contentType.value === CONTENT_TYPE_IMAGE
    ? '转载图片请注明来源、时间、地点，注明来源会更快地通过审核'
    : '转载视频请注明来源、时间、地点(例：转自https://www.xxxx.com/yyyy)，注明来源会更快地通过审核哦'
})

const introductionPlaceholder = computed(() => {
  return contentType.value === CONTENT_TYPE_IMAGE
    ? '填写更全面的相关信息，让更多的人能找到你的图片稿件吧'
    : '填写更全面的相关信息，让更多的人能找到你的视频吧'
})

mitter.on('startUpload', (fileName) => {
  contentType.value = CONTENT_TYPE_VIDEO
  startUpload.value = true
  nextTick(() => {
    formDataRef.value?.resetFields()
    formData.value = createDefaultFormData(fileName)
  })
})

const rules = {
  videoCover: [{ required: true, message: '封面不能为空' }],
  videoName: [{ required: true, message: '标题不能为空' }],
  postType: [{ required: true, message: '类型不能为空' }],
  originInfo: [{ required: true, message: '转载说明不能为空' }],
  categoryArray: [{ required: true, message: '分区不能为空' }],
  tags: [{ required: true, message: '标签不能为空' }],
  downloadPermission: [
    { required: true, message: '请选择视频下载设置' },
  ],
}

provide('cutImageCallback', ({ coverImage }) => {
  formData.value.videoCover = coverImage
})

const videoUploaderRef = ref()
const imagePostUploaderRef = ref()

const handleSwitchContentType = (type) => {
  if (contentType.value === type) {
    return
  }
  contentType.value = type
  formData.value = createDefaultFormData()
}

const handleImageListChange = (list) => {
  imagePostList.value = list || []
  if (contentType.value !== CONTENT_TYPE_IMAGE) {
    return
  }
  if (imagePostList.value.length === 0) {
    if (!videoId.value) {
      startImagePost.value = false
    }
    return
  }
  if (!startImagePost.value) {
    startImagePost.value = true
    const firstImageName = imagePostList.value[0]?.fileName || ''
    nextTick(() => {
      formDataRef.value?.resetFields()
      formData.value = createDefaultFormData(firstImageName)
    })
  }
}

const submitForm = () => {
  formDataRef.value.validate(async (valid) => {
    if (!valid) {
      return
    }

    let params = {}
    Object.assign(params, formData.value)
    params.contentType = contentType.value

    if (contentType.value === CONTENT_TYPE_VIDEO) {
      const uploadFileList = videoUploaderRef.value.getUploadFileList()
      if (!uploadFileList) {
        return
      }
      params.uploadFileList = JSON.stringify(uploadFileList)
    } else {
      const imageList = imagePostUploaderRef.value.getImageList()
      if (!imageList || imageList.length === 0) {
        proxy.Message.warning('请至少上传一张图片')
        return
      }
      params.imageList = JSON.stringify(imageList)
      params.downloadPermission = 1
      delete params.interactionArray
    }

    params.pCategoryId = params.categoryArray[0]
    if (params.categoryArray.length > 1) {
      params.categoryId = params.categoryArray[1]
    }
    delete params.categoryArray

    if (params.interactionArray) {
      params.interaction = params.interactionArray.join(',')
      delete params.interactionArray
    }
    if (params.videoCover instanceof File) {
      const videoCover = await uploadImage(params.videoCover)
      if (!videoCover) {
        return
      }
      params.videoCover = videoCover
    }
    let result = await proxy.Request({
      url: proxy.Api.postVideo,
      showLoading: true,
      params,
    })
    if (!result) {
      return
    }
    proxy.Message.success('发布成功')
    router.push('/ucenter/video')
  })
}

const init = async () => {
  if (!videoId.value) {
    contentType.value = CONTENT_TYPE_VIDEO
    startUpload.value = false
    startImagePost.value = false
    nextTick(() => {
      videoUploaderRef.value?.initUploader(startUpload.value, [])
      imagePostUploaderRef.value?.initUploader([])
    })
    return
  }

  let result = await proxy.Request({
    url: proxy.Api.getVideoByVideoId,
    params: {
      videoId: videoId.value,
    },
  })
  if (!result) {
    return
  }
  formData.value = result.data.videoInfo
  contentType.value = formData.value.contentType ?? CONTENT_TYPE_VIDEO
  startUpload.value = contentType.value === CONTENT_TYPE_VIDEO
  startImagePost.value = contentType.value === CONTENT_TYPE_IMAGE
  formData.value.downloadPermission = formData.value.downloadPermission ?? 1
  formData.value.tags = formData.value.tags ? formData.value.tags.split(',') : []
  formData.value.categoryArray = []
  if (formData.value.pCategoryId) {
    formData.value.categoryArray.push(formData.value.pCategoryId)
  }
  if (formData.value.categoryId) {
    formData.value.categoryArray.push(formData.value.categoryId)
  }
  formData.value.interactionArray = formData.value.interaction
    ? formData.value.interaction.split(',')
    : []

  nextTick(() => {
    if (contentType.value === CONTENT_TYPE_VIDEO) {
      videoUploaderRef.value?.initUploader(
        startUpload.value,
        result.data.videoInfoFileList
      )
      return
    }
    imagePostUploaderRef.value?.initUploader(result.data.videoInfoFileList)
  })
}

watch(
  () => route.query.videoId,
  (newVal) => {
    videoId.value = newVal
    init()
  },
  { immediate: true, deep: true }
)

const reload = () => {}
onMounted(() => {
  window.addEventListener('beforeunload', reload)
})

onUnmounted(() => {
  mitter.off('startUpload')

  window.removeEventListener('beforeunload', reload)
})
</script>

<style lang="scss" scoped>
.upload-video-panel {
  background: #fff;
  padding: 20px;
}
.post-type-tabs {
  display: flex;
  align-items: center;
  height: 56px;
  border-bottom: 1px solid #e5e5e5;
  margin: -20px -20px 20px;
  padding: 0 40px;
  .tab-item {
    height: 56px;
    line-height: 56px;
    margin-right: 44px;
    cursor: pointer;
    color: #333;
    border-bottom: 3px solid transparent;
    font-size: 15px;
    &:hover {
      color: #00a1d6;
    }
  }
  .active {
    color: #00a1d6;
    font-weight: bold;
    border-bottom-color: #00a1d6;
  }
}
.video-form {
  padding-right: 200px;
}
:deep(.download-setting-item .el-form-item__label) {
  white-space: nowrap;
}
</style>
