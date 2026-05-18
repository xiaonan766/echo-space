<template>
  <Dialog
    :show="dialogConfig.show"
    :title="dialogConfig.title"
    :buttons="dialogConfig.buttons"
    width="920px"
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
      <el-row :gutter="12">
        <el-col :span="12">
          <el-form-item label="开售时间" prop="saleStartTime">
            <el-date-picker
              v-model="formData.saleStartTime"
              type="datetime"
              value-format="YYYY-MM-DD HH:mm:ss"
              format="YYYY-MM-DD HH:mm:ss"
              clearable
              placeholder="不填写表示立即开售"
              class="form-date"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
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
        </el-col>
      </el-row>

      <el-form-item label="商品规格">
        <div class="sku-section">
          <div class="sku-toolbar">
            <div class="sku-summary">
              <span>共 {{ formData.skuList.length }} 个规格</span>
              <span>启用 {{ activeSkuCount }} 个</span>
              <span>总库存 {{ totalStockCount }}</span>
            </div>
            <el-button type="primary" size="small" :disabled="!canEditPrice" @click="addSku">
              添加规格
            </el-button>
          </div>
          <el-table :data="formData.skuList" border size="small" class="sku-table">
            <el-table-column label="规格名称" min-width="190">
              <template #default="{ row }">
                <el-input
                  v-model="row.skuName"
                  maxlength="80"
                  placeholder="如：白色 / M码"
                />
              </template>
            </el-table-column>
            <el-table-column label="单价" width="150">
              <template #default="{ row }">
                <el-input-number
                  v-model="row.price"
                  :min="0.01"
                  :precision="2"
                  :step="1"
                  :disabled="!canEditPrice"
                  controls-position="right"
                  class="table-number"
                />
              </template>
            </el-table-column>
            <el-table-column label="总库存" width="150">
              <template #default="{ row }">
                <el-input-number
                  v-model="row.totalStock"
                  :min="getMinSkuStock(row)"
                  :precision="0"
                  :step="1"
                  controls-position="right"
                  class="table-number"
                />
              </template>
            </el-table-column>
            <el-table-column label="占用" width="130">
              <template #default="{ row }">
                <div class="sku-occupied">
                  <span>锁定 {{ row.lockedStock || 0 }}</span>
                  <span>已售 {{ row.soldStock || 0 }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="可售" width="80">
              <template #default="{ row }">
                {{ getAvailableStock(row) }}
              </template>
            </el-table-column>
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-switch
                  v-model="row.status"
                  :active-value="1"
                  :inactive-value="0"
                  active-text="启用"
                  inactive-text="停用"
                  inline-prompt
                />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="80">
              <template #default="{ $index, row }">
                <el-button
                  v-if="!row.skuId"
                  link
                  type="danger"
                  @click="removeNewSku($index)"
                >
                  移除
                </el-button>
                <span v-else class="saved-text">已保存</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
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
          每个规格的总库存不能小于该规格的锁定库存与已售库存之和；已有规格请使用“停用”保留历史数据。
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

const createDefaultSku = () => ({
  skuId: '',
  skuName: '默认规格',
  price: 0.01,
  totalStock: 0,
  lockedStock: 0,
  soldStock: 0,
  status: 1,
})

const createDefaultFormData = () => ({
  productId: '',
  productName: '',
  coverUrl: '',
  description: '',
  lastOffShelfTime: '',
  originalStatus: '',
  saleStartTime: '',
  status: 1,
  recommendStatus: 0,
  sort: 0,
  skuList: [createDefaultSku()],
})

const formData = ref(createDefaultFormData())
const formDataRef = ref()

const activeSkuCount = computed(() => {
  return formData.value.skuList.filter((item) => Number(item.status) === 1).length
})

const totalStockCount = computed(() => {
  return formData.value.skuList.reduce((total, item) => total + Number(item.totalStock || 0), 0)
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
    return '当前商品仍处于上架状态，需先下架并等待30分钟后才能修改价格或新增规格。'
  }
  const remainMinutes = getPriceEditRemainMinutes()
  if (remainMinutes > 0) {
    return `商品下架未满30分钟，还需等待约 ${remainMinutes} 分钟才能修改价格或新增规格。`
  }
  return '当前商品暂不可修改价格或新增规格。'
})

const rules = {
  productName: [{ required: true, message: '请输入商品名称' }],
  status: [{ required: true, message: '请选择上架状态' }],
  recommendStatus: [{ required: true, message: '请选择推荐状态' }],
}

const showEdit = async (row = {}) => {
  dialogConfig.value.show = true
  dialogConfig.value.title = row.productId ? '编辑周边' : '新增周边'
  await nextTick()
  formDataRef.value?.resetFields()

  if (!row.productId) {
    formData.value = createDefaultFormData()
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
  const skuList = Array.isArray(result.data.skuList) && result.data.skuList.length > 0
    ? result.data.skuList.map(normalizeSku)
    : [normalizeSku(result.data)]

  formData.value = {
    ...createDefaultFormData(),
    ...result.data,
    status: Number(result.data.status),
    originalStatus: Number(result.data.status),
    recommendStatus: Number(result.data.recommendStatus),
    sort: Number(result.data.sort || 0),
    skuList,
  }
}

defineExpose({
  showEdit,
})

const emit = defineEmits(['reload'])

const addSku = () => {
  if (!canEditPrice.value) {
    proxy.Message.warning('需先下架并等待30分钟后才能新增规格')
    return
  }
  formData.value.skuList.push({
    ...createDefaultSku(),
    skuName: '',
  })
}

const removeNewSku = (index) => {
  if (formData.value.skuList.length <= 1) {
    proxy.Message.warning('至少保留一个规格')
    return
  }
  formData.value.skuList.splice(index, 1)
}

const submitForm = async () => {
  formDataRef.value.validate(async (valid) => {
    if (!valid || !validateSkuList()) {
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
    params.skuList = JSON.stringify(
      formData.value.skuList.map((item) => ({
        skuId: item.skuId || 0,
        skuName: String(item.skuName || '').trim(),
        price: Number(item.price || 0),
        totalStock: Number(item.totalStock || 0),
        status: Number(item.status),
      }))
    )

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

const validateSkuList = () => {
  const skuList = formData.value.skuList || []
  if (skuList.length === 0) {
    proxy.Message.warning('请至少添加一个商品规格')
    return false
  }

  const nameSet = new Set()
  let activeCount = 0
  for (const item of skuList) {
    const skuName = String(item.skuName || '').trim()
    if (!skuName) {
      proxy.Message.warning('请输入规格名称')
      return false
    }
    if (nameSet.has(skuName)) {
      proxy.Message.warning('规格名称不能重复')
      return false
    }
    nameSet.add(skuName)
    if (Number(item.price) <= 0) {
      proxy.Message.warning('请输入正确的规格单价')
      return false
    }
    if (Number(item.totalStock) < getMinSkuStock(item)) {
      proxy.Message.warning(`规格【${skuName}】总库存不能小于 ${getMinSkuStock(item)}`)
      return false
    }
    if (Number(item.status) === 1) {
      activeCount++
    }
  }
  if (activeCount === 0) {
    proxy.Message.warning('请至少启用一个商品规格')
    return false
  }
  return true
}

const normalizeSku = (sku = {}) => {
  return {
    skuId: sku.skuId || '',
    skuName: sku.skuName || '默认规格',
    price: Number(sku.price || 0.01),
    totalStock: Number(sku.totalStock || 0),
    lockedStock: Number(sku.lockedStock || 0),
    soldStock: Number(sku.soldStock || 0),
    status: Number(sku.status ?? 1),
  }
}

const getMinSkuStock = (row) => {
  return Number(row.lockedStock || 0) + Number(row.soldStock || 0)
}

const getAvailableStock = (row) => {
  return Math.max(Number(row.totalStock || 0) - getMinSkuStock(row), 0)
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
.sku-section {
  width: 100%;
}
.sku-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}
.sku-summary {
  display: flex;
  gap: 16px;
  color: var(--text2);
}
.sku-table {
  width: 100%;
}
.table-number {
  width: 100%;
}
.sku-occupied {
  display: flex;
  flex-direction: column;
  gap: 3px;
  color: var(--text3);
  line-height: 18px;
}
.saved-text {
  color: var(--text3);
}
</style>
