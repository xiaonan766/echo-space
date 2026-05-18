<template>
  <Dialog
    :show="dialogConfig.show"
    :title="dialogConfig.title"
    :buttons="dialogConfig.buttons"
    width="640px"
    :showCancel="true"
    @close="dialogConfig.show = false"
  >
    <el-form
      :model="formData"
      :rules="rules"
      ref="formDataRef"
      label-width="90px"
      @submit.prevent
    >
      <el-form-item label="商品名称" prop="productName">
        <el-input
          v-model="formData.productName"
          clearable
          maxlength="100"
          show-word-limit
          placeholder="请输入周边商品名称"
        />
      </el-form-item>
      <el-form-item label="封面图" prop="coverUrl">
        <ImageUpload v-model="formData.coverUrl"></ImageUpload>
      </el-form-item>
      <el-form-item label="商品描述" prop="description">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="4"
          maxlength="500"
          show-word-limit
          placeholder="请输入商品描述"
        />
      </el-form-item>
      <el-row :gutter="12">
        <el-col :span="12">
          <el-form-item label="单价" prop="price">
            <el-input-number
              v-model="formData.price"
              :min="0.01"
              :precision="2"
              :step="1"
              :disabled="!canEditPrice"
              controls-position="right"
              class="form-number"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="库存数量" prop="totalStock">
            <el-input-number
              v-model="formData.totalStock"
              :min="0"
              :precision="0"
              :step="1"
              controls-position="right"
              class="form-number"
            />
          </el-form-item>
        </el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :span="12">
          <el-form-item label="上架状态" prop="status">
            <el-radio-group v-model="formData.status">
              <el-radio :label="1">上架</el-radio>
              <el-radio :label="0">下架</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="热门推荐" prop="recommendStatus">
            <el-radio-group v-model="formData.recommendStatus">
              <el-radio :label="1">推荐</el-radio>
              <el-radio :label="0">不推荐</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="开售时间" prop="saleStartTime">
        <el-date-picker
          v-model="formData.saleStartTime"
          type="datetime"
          value-format="YYYY-MM-DD HH:mm:ss"
          format="YYYY-MM-DD HH:mm:ss"
          clearable
          placeholder="不填表示立即开售"
          class="form-date"
        />
      </el-form-item>
      <el-form-item label="排序" prop="sort">
        <el-input-number
          v-model="formData.sort"
          :min="0"
          :precision="0"
          :step="1"
          controls-position="right"
          class="form-number"
        />
      </el-form-item>
      <el-alert
        v-if="formData.productId && !canEditPrice"
        :closable="false"
        type="warning"
        show-icon
        class="stock-tip"
      >
        <template #title>
          {{ priceEditTip }}
        </template>
      </el-alert>
      <el-alert
        v-if="formData.productId"
        :closable="false"
        type="info"
        show-icon
        class="stock-tip"
      >
        <template #title>
          当前已售 {{ formData.soldStock || 0 }}，锁定 {{ formData.lockedStock || 0 }}，总库存不能小于
          {{ minEditableStock }}。
        </template>
      </el-alert>
    </el-form>
  </Dialog>
</template>

<script setup>
import ImageUpload from '@/components/ImageUpload.vue'
import { computed, getCurrentInstance, nextTick, ref } from 'vue'
import { uploadImage } from '@/utils/Api.js'

const { proxy } = getCurrentInstance()

const dialogConfig = ref({
  show: false,
  title: '新增周边',
  buttons: [
    {
      type: 'primary',
      text: '确定',
      click: () => {
        submitForm()
      },
    },
  ],
})

const defaultFormData = {
  productId: '',
  productName: '',
  coverUrl: '',
  description: '',
  price: 0.01,
  totalStock: 0,
  lockedStock: 0,
  soldStock: 0,
  lastOffShelfTime: '',
  originalStatus: '',
  saleStartTime: '',
  status: 1,
  recommendStatus: 0,
  sort: 0,
}

const formData = ref({ ...defaultFormData })
const formDataRef = ref()

const minEditableStock = computed(() => {
  return Number(formData.value.lockedStock || 0) + Number(formData.value.soldStock || 0)
})

const canEditPrice = computed(() => {
  if (!formData.value.productId) {
    return true
  }
  if (Number(formData.value.originalStatus) === 1) {
    return false
  }
  if (!formData.value.lastOffShelfTime) {
    return true
  }
  return Date.now() - parseDateTime(formData.value.lastOffShelfTime).getTime() >= 30 * 60 * 1000
})

const priceEditTip = computed(() => {
  if (Number(formData.value.originalStatus) === 1) {
    return '当前商品仍处于上架状态，需先下架并等待30分钟后才能修改价格。'
  }
  const remainMinutes = getPriceEditRemainMinutes()
  if (remainMinutes > 0) {
    return `商品下架未满30分钟，还需等待约 ${remainMinutes} 分钟才能修改价格。`
  }
  return '当前商品暂不可修改价格。'
})

const validateStock = (rule, value, callback) => {
  if (value == null || value < 0) {
    callback(new Error('请输入正确的库存数量'))
    return
  }
  if (formData.value.productId && value < minEditableStock.value) {
    callback(new Error(`总库存不能小于 ${minEditableStock.value}`))
    return
  }
  callback()
}

const rules = {
  productName: [{ required: true, message: '请输入商品名称' }],
  price: [{ required: true, message: '请输入商品单价' }],
  totalStock: [{ required: true, validator: validateStock }],
  status: [{ required: true, message: '请选择上架状态' }],
  recommendStatus: [{ required: true, message: '请选择推荐状态' }],
}

const showEdit = async (row = {}) => {
  dialogConfig.value.show = true
  dialogConfig.value.title = row.productId ? '编辑周边' : '新增周边'
  await nextTick()
  formDataRef.value?.resetFields()

  if (!row.productId) {
    formData.value = { ...defaultFormData }
    return
  }

  let result = await proxy.Request({
    url: proxy.Api.getPeripheral,
    params: {
      productId: row.productId,
    },
  })
  if (!result) {
    dialogConfig.value.show = false
    return
  }
  formData.value = {
    ...defaultFormData,
    ...result.data,
    price: Number(result.data.price || 0),
    totalStock: Number(result.data.totalStock || 0),
    lockedStock: Number(result.data.lockedStock || 0),
    soldStock: Number(result.data.soldStock || 0),
    status: Number(result.data.status),
    originalStatus: Number(result.data.status),
    recommendStatus: Number(result.data.recommendStatus),
    sort: Number(result.data.sort || 0),
  }
}

defineExpose({
  showEdit,
})

const emit = defineEmits(['reload'])

const submitForm = async () => {
  formDataRef.value.validate(async (valid) => {
    if (!valid) {
      return
    }

    let params = { ...formData.value }
    if (params.coverUrl instanceof File) {
      const uploadedCover = await uploadImage(params.coverUrl)
      if (!uploadedCover) {
        return
      }
      params.coverUrl = uploadedCover
    }

    let result = await proxy.Request({
      url: proxy.Api.savePeripheral,
      params,
    })
    if (!result) {
      return
    }

    dialogConfig.value.show = false
    proxy.Message.success('保存成功')
    emit('reload')
  })
}

const parseDateTime = (value) => {
  return new Date(String(value).replace(/-/g, '/'))
}

const getPriceEditRemainMinutes = () => {
  if (!formData.value.lastOffShelfTime) {
    return 0
  }
  const elapsed = Date.now() - parseDateTime(formData.value.lastOffShelfTime).getTime()
  const remain = 30 * 60 * 1000 - elapsed
  return remain > 0 ? Math.ceil(remain / 60000) : 0
}
</script>

<style lang="scss" scoped>
.form-number {
  width: 100%;
}
.form-date {
  width: 100%;
}
.stock-tip {
  margin-top: 4px;
}
</style>
