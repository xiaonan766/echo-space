<template>
  <div class="image-post-uploader">
    <el-upload
      class="image-upload-drop"
      drag
      multiple
      :show-file-list="false"
      :http-request="addImage"
      :accept="proxy.imageAccept"
    >
      <div class="upload-handler">
        <div class="iconfont icon-upload"></div>
        <div class="info">点击上传或将图片拖拽到此区域</div>
        <div class="upload-btn">上传图片</div>
        <div class="tips">支持 jpg、png、gif、bmp、webp，最多 9 张</div>
      </div>
    </el-upload>

    <div
      class="image-list"
      v-if="imageList.length > 0"
      v-draggable="[imageList, { animation: 150 }]"
    >
      <div class="image-item" v-for="(item, index) in imageList" :key="item.uid">
        <Cover :source="item.sourceName" :width="140" :preview="!!item.sourceName"></Cover>
        <div class="image-info">
          <div class="image-name" :title="item.fileName">{{ item.fileName }}</div>
          <div class="image-status" :class="item.status">
            {{ IMAGE_STATUS[item.status] }}
          </div>
        </div>
        <div class="image-index">图{{ index + 1 }}</div>
        <div class="iconfont icon-del delete-btn" @click="deleteImage(index)"></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { getCurrentInstance, ref } from 'vue'
import { vDraggable } from 'vue-draggable-plus'
import { uploadImage } from '@/utils/Api.js'

const { proxy } = getCurrentInstance()

const MAX_IMAGE_COUNT = 9
const IMAGE_STATUS = {
  uploading: '上传中',
  success: '上传完成',
  fail: '上传失败',
}

const imageList = ref([])
const emit = defineEmits(['change'])

const addImage = async (uploadFile) => {
  const file = uploadFile.file
  if (imageList.value.length >= MAX_IMAGE_COUNT) {
    proxy.Message.warning(`图片投稿最多支持 ${MAX_IMAGE_COUNT} 张图片`)
    return
  }

  const fileName = proxy.Utils.getFileName(file.name)
  const imageItem = {
    uid: file.uid || `${Date.now()}-${imageList.value.length}`,
    fileName,
    sourceName: '',
    status: 'uploading',
  }
  imageList.value.push(imageItem)
  emitChange()

  const sourceName = await uploadImage(file)
  if (!sourceName) {
    imageItem.status = 'fail'
    emitChange()
    return
  }
  imageItem.sourceName = sourceName
  imageItem.status = 'success'
  emitChange()
}

const deleteImage = (index) => {
  imageList.value.splice(index, 1)
  emitChange()
}

const emitChange = () => {
  emit('change', getImageList(false))
}

const getImageList = (showMessage = true) => {
  const failItem = imageList.value.find((item) => item.status === 'fail')
  if (failItem) {
    if (showMessage) {
      proxy.Message.warning('请删除上传失败的图片')
    }
    return null
  }
  const uploadingItem = imageList.value.find((item) => item.status === 'uploading')
  if (uploadingItem) {
    if (showMessage) {
      proxy.Message.warning('图片还未上传完成，无法提交')
    }
    return null
  }
  return imageList.value
    .filter((item) => item.sourceName)
    .map((item) => ({
      fileId: item.fileId,
      sourceName: item.sourceName,
      fileName: item.fileName,
    }))
}

const initUploader = (list = []) => {
  imageList.value = list.map((item, index) => ({
    uid: item.fileId || `${item.filePath}-${index}`,
    fileId: item.fileId,
    sourceName: item.filePath,
    fileName: item.fileName,
    status: 'success',
  }))
  emitChange()
}

defineExpose({
  getImageList,
  initUploader,
})
</script>

<style lang="scss" scoped>
.image-post-uploader {
  .image-upload-drop {
    margin: 20px 200px;
    border: 2px dashed #d8d8d8;
    border-radius: 6px;
    :deep(.el-upload-dragger) {
      border: none;
    }
    .upload-handler {
      color: #999;
      padding: 45px 0;
      text-align: center;
      .icon-upload {
        font-size: 34px;
      }
      .info {
        margin: 16px 0;
      }
      .upload-btn {
        color: #fff;
        margin: 16px auto 10px;
        width: 200px;
        height: 44px;
        cursor: pointer;
        background: #00a1d6;
        border-radius: 4px;
        line-height: 44px;
      }
      .tips {
        font-size: 12px;
      }
    }
  }
  .image-list {
    margin: 0 200px 20px 20px;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 14px;
    .image-item {
      position: relative;
      background: #f6f7f8;
      border-radius: 6px;
      padding: 10px;
      min-width: 0;
      .image-info {
        margin-top: 8px;
        .image-name {
          overflow: hidden;
          white-space: nowrap;
          text-overflow: ellipsis;
        }
        .image-status {
          margin-top: 4px;
          font-size: 12px;
          color: #67c23a;
        }
        .uploading {
          color: #409eff;
        }
        .fail {
          color: #f56c6c;
        }
      }
      .image-index {
        position: absolute;
        left: 10px;
        top: 10px;
        padding: 2px 6px;
        border-radius: 0 0 4px 0;
        color: #fff;
        background: rgba(0, 0, 0, 0.6);
        font-size: 12px;
      }
      .delete-btn {
        position: absolute;
        top: 8px;
        right: 8px;
        cursor: pointer;
        color: #fff;
        padding: 4px;
        border-radius: 50%;
        background: rgba(0, 0, 0, 0.5);
      }
    }
  }
}
</style>
