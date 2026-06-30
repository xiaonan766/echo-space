<template>
  <div class="cabinet-panel" v-loading="loading">
    <div class="cabinet-title-panel">
      <div>
        <div class="cabinet-title">周边柜</div>
        <div class="cabinet-subtitle">展示已经拥有的周边商品</div>
      </div>
      <div class="cabinet-switch" v-if="dataSource.owner">
        <span>公开展示收藏柜</span>
        <el-switch
          v-model="dataSource.cabinetVisible"
          :loading="visibleUpdating"
          :disabled="visibleUpdating"
          @change="handleCabinetVisibleChange"
        ></el-switch>
      </div>
    </div>

    <div v-if="loadFailed" class="cabinet-empty">
      <NoData msg="周边柜加载失败，请稍后再试"></NoData>
    </div>
    <div v-else-if="showPrivateNotice" class="cabinet-empty">
      <NoData msg="空间主人隐藏了周边收藏柜"></NoData>
    </div>
    <div v-else-if="hasLoaded && dataSource.list && dataSource.list.length === 0" class="cabinet-empty">
      <NoData :msg="emptyMessage"></NoData>
    </div>
    <DataGridList
      v-else-if="hasLoaded"
      :dataSource="dataSource"
      :gridCount="4"
      @loadData="loadCabinet"
    >
      <template #default="{ data }">
        <div :class="['cabinet-card', data.hidden ? 'is-hidden' : '']" @click="jumpProduct(data)">
          <div class="cover" :style="getCoverStyle(data)">
            <span v-if="!data.coverUrl">暂无图片</span>
            <div class="hidden-badge" v-if="data.hidden">已隐藏</div>
          </div>
          <div class="card-info">
            <div class="product-name">{{ data.productName || '已失效周边' }}</div>
            <div class="sku-name">{{ data.skuName || '默认规格' }}</div>
            <div class="own-info">
              <span>拥有 {{ data.ownedQuantity || 0 }} 件</span>
              <span>{{ data.orderCount || 0 }} 笔订单</span>
            </div>
            <div class="bottom-row">
              <span class="buy-time">最近购买：{{ data.latestBuyTime || '-' }}</span>
              <el-button
                v-if="dataSource.owner"
                text
                class="visible-button"
                :loading="updatingSkuId === data.skuId"
                :disabled="updatingSkuId > 0"
                @click.stop="handleItemVisibleChange(data)"
              >
                {{ data.hidden ? '展示' : '隐藏' }}
              </el-button>
            </div>
          </div>
        </div>
      </template>
    </DataGridList>
  </div>
</template>

<script setup>
import { computed, getCurrentInstance, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const { proxy } = getCurrentInstance()
const route = useRoute()
const router = useRouter()

const loading = ref(false)
const hasLoaded = ref(false)
const loadFailed = ref(false)
const visibleUpdating = ref(false)
const updatingSkuId = ref(0)
const dataSource = ref({
  owner: false,
  cabinetVisible: false,
  list: [],
  pageNo: 1,
  pageSize: 12,
  pageTotal: 0,
  totalCount: 0,
})

const showPrivateNotice = computed(() => {
  return hasLoaded.value && !loadFailed.value && !dataSource.value.owner && dataSource.value.cabinetVisible === false
})

const emptyMessage = computed(() => {
  if (dataSource.value.owner) {
    return '你还没有购买周边'
  }
  return '空间主人还没有展示周边'
})

const loadCabinet = async () => {
  loading.value = true
  loadFailed.value = false
  const result = await proxy.Request({
    url: proxy.Api.uHomeLoadPeripheralCabinet,
    params: {
      userId: route.params.userId,
      pageNo: dataSource.value.pageNo || 1,
      pageSize: dataSource.value.pageSize || 12,
    },
    showError: false,
  })
  loading.value = false
  if (!result) {
    hasLoaded.value = true
    loadFailed.value = true
    dataSource.value = normalizeCabinetData(null)
    return
  }
  dataSource.value = normalizeCabinetData(result.data)
  hasLoaded.value = true
}

const handleCabinetVisibleChange = async (visible) => {
  if (visibleUpdating.value) {
    return
  }
  const oldVisible = !visible
  visibleUpdating.value = true
  const result = await proxy.Request({
    url: proxy.Api.uHomeUpdatePeripheralCabinetVisible,
    params: {
      visible: visible ? 1 : 0,
    },
    showLoading: true,
  })
  if (!result) {
    dataSource.value.cabinetVisible = oldVisible
    visibleUpdating.value = false
    return
  }
  visibleUpdating.value = false
  proxy.Message.success(visible ? '周边柜已公开展示' : '周边柜已隐藏')
}

const handleItemVisibleChange = async (item) => {
  if (!item?.skuId || updatingSkuId.value) {
    return
  }
  const nextVisible = item.hidden
  updatingSkuId.value = item.skuId
  const result = await proxy.Request({
    url: proxy.Api.uHomeUpdatePeripheralCabinetItemVisible,
    params: {
      skuId: item.skuId,
      visible: nextVisible ? 1 : 0,
    },
    showLoading: true,
  })
  updatingSkuId.value = 0
  if (!result) {
    return
  }
  item.hidden = !nextVisible
  proxy.Message.success(nextVisible ? '该周边已展示' : '该周边已隐藏')
}

const jumpProduct = (item) => {
  if (!item?.productId) {
    return
  }
  router.push(`/shop/peripheral/${item.productId}`)
}

const getCoverStyle = (item) => {
  if (!item?.coverUrl) {
    return {}
  }
  return {
    backgroundImage: `url(${buildImageUrl(item.coverUrl)})`,
  }
}

const buildImageUrl = (source) => {
  if (!source) {
    return ''
  }
  if (/^(https?:)?\/\//.test(source) || source.startsWith('data:') || source.startsWith('/')) {
    return source
  }
  return `${proxy.Api.sourcePath}${source}`
}

const normalizeCabinetData = (data) => {
  return {
    owner: !!data?.owner,
    cabinetVisible: !!data?.cabinetVisible,
    list: data?.list || [],
    pageNo: data?.pageNo || 1,
    pageSize: data?.pageSize || 12,
    pageTotal: data?.pageTotal || 0,
    totalCount: data?.totalCount || 0,
  }
}

watch(
  () => route.params.userId,
  () => {
    hasLoaded.value = false
    loadFailed.value = false
    dataSource.value.pageNo = 1
    loadCabinet()
  },
  { immediate: true }
)
</script>

<style lang="scss" scoped>
.cabinet-panel {
  padding: 20px;
  background: #fff;
  border-radius: 5px;
  min-height: 320px;
}

.cabinet-title-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 18px;
  .cabinet-title {
    font-size: 18px;
    color: var(--text);
  }
  .cabinet-subtitle {
    margin-top: 6px;
    color: var(--text3);
    font-size: 13px;
  }
}

.cabinet-switch {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text2);
  font-size: 13px;
}

.cabinet-empty {
  padding: 45px 0;
}

.cabinet-card {
  border: 1px solid #f0f1f4;
  border-radius: 6px;
  background: #fff;
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.18s ease, box-shadow 0.18s ease;
  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px rgba(16, 18, 24, 0.08);
    .product-name {
      color: #fb7299;
    }
  }
  &.is-hidden {
    .cover,
    .card-info {
      opacity: 0.62;
    }
  }
}

.cover {
  position: relative;
  width: 100%;
  aspect-ratio: 1 / 1;
  background: #edf0f5;
  background-size: cover;
  background-position: center;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text3);
}

.hidden-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 3px 8px;
  border-radius: 3px;
  background: rgba(0, 0, 0, 0.58);
  color: #fff;
  font-size: 12px;
}

.card-info {
  padding: 12px;
}

.product-name {
  height: 40px;
  color: var(--text);
  font-size: 14px;
  line-height: 20px;
  font-weight: 600;
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  word-break: break-word;
}

.sku-name {
  margin-top: 8px;
  color: var(--text2);
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.own-info {
  margin-top: 10px;
  display: flex;
  justify-content: space-between;
  gap: 10px;
  color: #fb7299;
  font-size: 13px;
}

.bottom-row {
  margin-top: 10px;
  min-height: 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--text3);
  font-size: 12px;
}

.buy-time {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.visible-button {
  flex: 0 0 auto;
  color: #fb7299;
}
</style>
