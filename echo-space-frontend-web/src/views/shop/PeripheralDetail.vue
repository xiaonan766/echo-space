<template>
  <div class="shop-detail-page">
    <div class="shop-header">
      <div class="header-inner">
        <div class="shop-brand" @click="goShopHome">
          <span class="brand-main">Echo Space</span>
          <span class="brand-sub">购物</span>
        </div>
        <button class="home-link iconfont icon-home" type="button" @click="goMainSite">
          主站
        </button>
        <div class="shop-search">
          <input
            v-model.trim="searchForm.keyword"
            placeholder="搜索商品名称"
            @keyup.enter="handleSearch"
          />
          <button class="iconfont icon-search" type="button" @click="handleSearch"></button>
        </div>
        <div class="user-area">
          <Avatar
            class="shop-avatar"
            :avatar="loginStore.userInfo.avatar"
            :userId="loginStore.userInfo.userId"
            :width="38"
            :lazy="false"
            @click="handleAvatarClick"
          ></Avatar>
          <button class="order-link iconfont icon-list" type="button" @click="handleOrderCenter">
            订单中心
          </button>
        </div>
      </div>
    </div>

    <main class="detail-main" v-loading="loading">
      <div class="detail-nav">
        <button type="button" @click="goPeripheralList">周边</button>
        <span>/</span>
        <span>商品详情</span>
      </div>

      <section class="detail-panel" v-if="product">
        <div class="cover-panel">
          <div class="cover-image" :style="coverStyle">
            <span v-if="!product.coverUrl">暂无图片</span>
          </div>
        </div>

        <div class="info-panel">
          <div :class="['status-tag', canBuy ? '' : 'disabled']">
            {{ product.saleStatusName || '在售' }}
          </div>
          <h1>{{ product.itemName || '未命名商品' }}</h1>
          <div class="price">{{ currentPriceText }}</div>

          <div class="meta-list">
            <div class="meta-row">
              <span class="label">开售时间</span>
              <span>{{ product.saleStartTime || '立即开售' }}</span>
            </div>
            <div class="meta-row">
              <span class="label">库存</span>
              <span>{{ currentStockText }}</span>
            </div>
          </div>

          <div class="sku-row">
            <span class="label">规格</span>
            <div class="sku-options">
              <button
                v-for="sku in skuList"
                :key="sku.skuId"
                :class="['sku-option', selectedSku?.skuId === sku.skuId ? 'active' : '']"
                :disabled="sku.availableStock <= 0"
                type="button"
                @click="handleSelectSku(sku)"
              >
                <span class="sku-name">{{ sku.skuName || '默认规格' }}</span>
                <span class="sku-extra">{{ sku.priceText }} · {{ sku.stockText }}</span>
              </button>
            </div>
          </div>

          <div class="quantity-row">
            <span class="label">数量</span>
            <el-input-number
              v-model="quantity"
              :min="1"
              :max="maxQuantity"
              :disabled="!canBuy"
              controls-position="right"
            ></el-input-number>
          </div>

          <el-button
            class="buy-button"
            type="primary"
            size="large"
            :disabled="!canBuy"
            @click="handleBuy"
          >
            {{ buyButtonText }}
          </el-button>
        </div>
      </section>

      <NoData v-else-if="!loading" :msg="emptyText"></NoData>

      <section class="description-section" v-if="product">
        <h2>商品详情</h2>
        <p v-if="product.description">{{ product.description }}</p>
        <NoData v-else msg="暂无商品描述"></NoData>
      </section>
    </main>
  </div>
</template>

<script setup>
import { computed, getCurrentInstance, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useLoginStore } from '@/stores/loginStore.js'
import { useNavAction } from '@/stores/navActionStore'

const { proxy } = getCurrentInstance()
const route = useRoute()
const router = useRouter()
const loginStore = useLoginStore()
const navActionStore = useNavAction()

const loading = ref(false)
const product = ref(null)
const selectedSku = ref(null)
const quantity = ref(1)
const searchForm = reactive({
  keyword: '',
})

const skuList = computed(() => {
  return product.value?.skuList || []
})

const currentSku = computed(() => {
  return selectedSku.value || skuList.value[0] || null
})

const currentPriceText = computed(() => {
  return currentSku.value?.priceText || product.value?.priceText || '价格待定'
})

const currentStockText = computed(() => {
  if (currentSku.value) {
    return currentSku.value.stockText || (currentSku.value.availableStock > 0 ? `库存 ${currentSku.value.availableStock}` : '暂无库存')
  }
  return product.value?.stockText || '暂无库存'
})

const maxQuantity = computed(() => {
  const stock = currentSku.value?.availableStock || 0
  return stock > 0 ? Math.min(stock, 99) : 1
})

const canBuy = computed(() => {
  return product.value?.saleStatus === 1 && currentSku.value?.saleStatus === 1 && currentSku.value?.availableStock > 0
})

const buyButtonText = computed(() => {
  if (!product.value) {
    return '立即购买'
  }
  if (product.value.saleStatus === 0) {
    return '暂未开售'
  }
  if (product.value.saleStatus === 2 || !currentSku.value || currentSku.value.availableStock <= 0) {
    return '已售罄'
  }
  return '立即购买'
})

const emptyText = computed(() => {
  return '周边商品不存在或已下架'
})

const coverStyle = computed(() => {
  const coverUrl = product.value?.coverUrl
  if (!coverUrl) {
    return {}
  }
  return {
    backgroundImage: `url(${buildImageUrl(coverUrl)})`,
  }
})

onMounted(() => {
  navActionStore.setShowHeader(false)
  navActionStore.setFixedHeader(false)
  navActionStore.setFixedCategory(false)
  navActionStore.setShowCategory(false)
  navActionStore.setForceFixedHeader(false)
})

const loadDetail = async () => {
  const productId = route.params.productId
  if (!productId) {
    product.value = null
    return
  }

  loading.value = true
  const result = await proxy.Request({
    url: proxy.Api.getPeripheralDetail,
    params: {
      productId,
    },
    showError: false,
  })
  loading.value = false

  product.value = result?.data || null
  selectedSku.value = getDefaultSku(product.value?.skuList || [])
  quantity.value = 1
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

const goShopHome = () => {
  router.push('/shop')
}

const goPeripheralList = () => {
  router.push({
    path: '/shop',
    query: {
      type: 'goods',
    },
  })
}

const goMainSite = () => {
  router.push('/')
}

const handleSearch = () => {
  router.push({
    path: '/shop',
    query: {
      type: 'goods',
      keyword: searchForm.keyword || undefined,
    },
  })
}

const handleAvatarClick = () => {
  if (Object.keys(loginStore.userInfo).length === 0) {
    loginStore.setLogin(true)
  }
}

const handleOrderCenter = () => {
  if (Object.keys(loginStore.userInfo).length === 0) {
    loginStore.setLogin(true)
    return
  }
  proxy.Message.warning('订单中心待接入')
}

const handleSelectSku = (sku) => {
  if (sku.availableStock <= 0) {
    return
  }
  selectedSku.value = sku
  quantity.value = 1
}

const handleBuy = () => {
  proxy.Message.warning('订单功能待接入')
}

const getDefaultSku = (list) => {
  return list.find((item) => item.saleStatus === 1 && item.availableStock > 0) || list[0] || null
}

watch(
  () => route.params.productId,
  () => {
    loadDetail()
  },
  { immediate: true }
)
</script>

<style lang="scss" scoped>
.shop-detail-page {
  min-height: 100vh;
  background: #f5f6f8;
  color: var(--text);
}

.shop-header {
  position: sticky;
  top: 0;
  z-index: 1000;
  height: 58px;
  background: #fff;
  box-shadow: 0 1px 8px rgba(0, 0, 0, 0.08);
  .header-inner {
    max-width: 1250px;
    height: 100%;
    margin: 0 auto;
    display: grid;
    grid-template-columns: 170px 90px minmax(320px, 1fr) 210px;
    align-items: center;
    column-gap: 18px;
  }
}

.shop-brand {
  cursor: pointer;
  display: flex;
  align-items: baseline;
  white-space: nowrap;
  .brand-main {
    color: #fb7299;
    font-size: 20px;
    font-weight: 700;
  }
  .brand-sub {
    margin-left: 6px;
    font-size: 15px;
    color: #222;
  }
}

.home-link,
.order-link,
.shop-search button,
.detail-nav button {
  border: none;
  background: transparent;
  padding: 0;
  font: inherit;
  cursor: pointer;
}

.home-link {
  color: #61666d;
  text-align: left;
  &::before {
    margin-right: 6px;
    color: #fb7299;
  }
  &:hover {
    color: #fb7299;
  }
}

.shop-search {
  height: 38px;
  display: flex;
  align-items: center;
  border-radius: 20px;
  background: #f1f2f3;
  border: 1px solid transparent;
  overflow: hidden;
  &:focus-within {
    background: #fff;
    border-color: #fb7299;
  }
  input {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
    padding: 0 16px;
    color: var(--text);
  }
  button {
    width: 42px;
    height: 100%;
    color: #61666d;
    &:hover {
      color: #fb7299;
    }
  }
}

.user-area {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 18px;
}

.shop-avatar {
  cursor: pointer;
}

.order-link {
  color: #61666d;
  white-space: nowrap;
  &::before {
    margin-right: 6px;
  }
  &:hover {
    color: #fb7299;
  }
}

.detail-main {
  max-width: 1250px;
  margin: 0 auto;
  padding: 28px 0 56px;
}

.detail-nav {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 18px;
  color: var(--text3);
  button {
    color: #61666d;
    &:hover {
      color: #fb7299;
    }
  }
}

.detail-panel {
  display: grid;
  grid-template-columns: 520px minmax(0, 1fr);
  gap: 46px;
  padding: 34px;
  border-radius: 6px;
  background: #fff;
  box-shadow: 0 8px 28px rgba(16, 18, 24, 0.05);
}

.cover-panel {
  min-width: 0;
}

.cover-image {
  width: 100%;
  aspect-ratio: 1 / 1;
  border-radius: 6px;
  background-color: #edf0f5;
  background-size: cover;
  background-position: center;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text3);
  overflow: hidden;
}

.info-panel {
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  .status-tag {
    margin-bottom: 14px;
    padding: 4px 10px;
    border-radius: 3px;
    color: #fb7299;
    border: 1px solid #fb7299;
    line-height: 20px;
    &.disabled {
      color: #909399;
      border-color: #c8c9cc;
    }
  }
  h1 {
    width: 100%;
    margin: 0;
    color: #222;
    font-size: 26px;
    line-height: 38px;
    font-weight: 700;
    word-break: break-word;
  }
}

.price {
  margin-top: 24px;
  color: #fb7299;
  font-size: 34px;
  line-height: 42px;
  font-weight: 700;
}

.meta-list {
  width: 100%;
  margin-top: 24px;
  padding: 16px 18px;
  border-radius: 6px;
  background: #f7f8fa;
}

.meta-row,
.sku-row,
.quantity-row {
  display: flex;
  align-items: center;
  min-height: 34px;
  color: #61666d;
  .label {
    flex: 0 0 82px;
    color: var(--text3);
  }
}

.sku-row {
  width: 100%;
  align-items: flex-start;
  margin-top: 24px;
}

.sku-options {
  flex: 1;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.sku-option {
  min-width: 148px;
  max-width: 220px;
  min-height: 54px;
  padding: 8px 12px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  background: #fff;
  color: #222;
  text-align: left;
  cursor: pointer;
  .sku-name,
  .sku-extra {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sku-extra {
    margin-top: 4px;
    color: var(--text3);
    font-size: 12px;
  }
  &:hover {
    border-color: #fb7299;
  }
  &.active {
    border-color: #fb7299;
    color: #fb7299;
    background: #fff5f8;
  }
  &:disabled {
    cursor: not-allowed;
    color: #c0c4cc;
    background: #f5f7fa;
    border-color: #e4e7ed;
  }
}

.quantity-row {
  margin-top: 24px;
  gap: 12px;
}

.buy-button {
  width: 220px;
  margin-top: 34px;
  border-color: #fb7299;
  background: #fb7299;
  &:hover,
  &:focus {
    border-color: #ff8aac;
    background: #ff8aac;
  }
  &.is-disabled {
    border-color: #c8c9cc;
    background: #c8c9cc;
  }
}

.description-section {
  margin-top: 26px;
  padding: 28px 34px 34px;
  border-radius: 6px;
  background: #fff;
  box-shadow: 0 8px 28px rgba(16, 18, 24, 0.04);
  h2 {
    margin: 0 0 18px;
    font-size: 22px;
    line-height: 30px;
  }
  p {
    margin: 0;
    color: #61666d;
    line-height: 28px;
    white-space: pre-wrap;
    word-break: break-word;
  }
}
</style>
