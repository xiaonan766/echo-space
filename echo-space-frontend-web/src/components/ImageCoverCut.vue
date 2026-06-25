<template>
  <Dialog
    :show="dialogConfig.show"
    :title="dialogConfig.title"
    :buttons="dialogConfig.buttons"
    width="min(1000px, calc(100vw - 32px))"
    :top="30"
    :append-to-body="true"
    @close="dialogConfig.show = false"
  >
    <div class="cut-image-panel">
      <VueCropper
        ref="cropperRef"
        class="cropper"
        :img="sourceImage"
        outputType="png"
        :autoCrop="true"
        :autoCropWidth="props.cutWidth"
        :autoCropHeight="Math.round(props.cutWidth * props.scale)"
        :fixed="true"
        :fixedNumber="[1, props.scale]"
        :centerBox="true"
        :full="false"
        :fixedBox="true"
        mode="100%"
        @realTime="preview"
      />
      <div class="preview-panel">
        <div class="preview-image">
          <img v-if="previewsImage" :src="previewsImage" />
        </div>
        <el-upload
          :multiple="false"
          :show-file-list="false"
          :http-request="selectFile"
          :accept="proxy.imageAccept"
        >
          <el-button class="select-btn" type="primary">选择图片</el-button>
        </el-upload>
      </div>
    </div>
    <div class="info">
      建议上传至少 {{ props.cutWidth }}*{{ Math.round(props.cutWidth * props.scale) }} 的图片
    </div>
  </Dialog>
</template>

<script setup>
import 'vue-cropper/dist/index.css'
import { VueCropper } from 'vue-cropper'
import { ref, getCurrentInstance, nextTick, inject } from 'vue'

const { proxy } = getCurrentInstance()

const props = defineProps({
  cutWidth: {
    type: Number,
    default: 400,
  },
  scale: {
    type: Number,
    default: 0.5,
  },
})

const dialogConfig = ref({
  show: false,
  title: '上传图片',
  buttons: [
    {
      type: 'primary',
      text: '确定',
      click: () => {
        cutImage()
      },
    },
  ],
})

const cropperRef = ref()
const previewsImage = ref()
const sourceImage = ref()

const preview = () => {
  cropperRef.value?.getCropData((data) => {
    previewsImage.value = data
  })
}

const selectFile = ({ file }) => {
  const img = new FileReader()
  img.readAsDataURL(file)
  img.onload = ({ target }) => {
    sourceImage.value = target.result
  }
}

const show = () => {
  dialogConfig.value.show = true
  sourceImage.value = ''
  nextTick(() => {
    previewsImage.value = ''
  })
}

defineExpose({
  show,
})

const cutImageCallback = inject('cutImageCallback')
const cutImage = () => {
  const cropW = Math.round(cropperRef.value?.cropW || 0)
  const cropH = Math.round(cropperRef.value?.cropH || 0)
  const minHeight = Math.round(props.cutWidth * props.scale)

  if (cropW === 0 || cropH === 0) {
    proxy.Message.warning('请选择图片')
    return
  }
  if (cropW < props.cutWidth || cropH < minHeight) {
    proxy.Message.warning(`图片尺寸至少满足 ${props.cutWidth}*${minHeight}`)
    return
  }

  cropperRef.value.getCropBlob((blob) => {
    const file = new File(
      [blob],
      'temp.' + blob.type.substring(blob.type.indexOf('/') + 1),
      { type: blob.type }
    )
    dialogConfig.value.show = false
    cutImageCallback({
      coverImage: file,
    })
  })
}
</script>

<style lang="scss" scoped>
.cut-image-panel {
  display: flex;
  min-height: 0;

  .cropper {
    flex: 1;
    height: min(500px, calc(100vh - 280px));
    min-height: 280px;
  }

  .preview-panel {
    width: 200px;
    margin-left: 20px;
    text-align: center;

    .preview-image {
      width: 100%;
      height: 200px;
      background: #f6f6f6;
      display: flex;
      align-items: center;
      justify-content: center;
      overflow: hidden;
    }

    img {
      width: 100%;
    }
  }

  .select-btn {
    margin-top: 20px;
  }
}

.info {
  color: #6b6b6b;
  margin-top: 10px;
}
</style>
