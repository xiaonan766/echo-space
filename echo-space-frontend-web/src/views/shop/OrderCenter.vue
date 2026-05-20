<template>
  <div class="shop-order-page">
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
          <button class="order-link iconfont icon-list active" type="button">
            订单中心
          </button>
        </div>
      </div>
    </div>

    <main class="order-main">
      <div class="order-nav">
        <button type="button" @click="goShopHome">购物</button>
        <span>/</span>
        <span>订单中心</span>
      </div>

      <section class="result-panel" v-if="activeOrder" v-loading="detailLoading">
        <div :class="['result-status', orderStatusClass(activeOrder)]">
          {{ activeOrder.orderStatusName }}
        </div>
        <div class="result-info">
          <h1>{{ activeOrder.productName }}</h1>
          <div class="order-no">订单号：{{ activeOrder.orderNo }}</div>
          <div class="result-meta">
            <span>{{ activeOrder.skuName }}</span>
            <span>数量 {{ activeOrder.quantity }}</span>
            <span>{{ activeOrder.totalAmountText }}</span>
          </div>
          <p v-if="activeOrder.orderStatus === 0">库存正在异步锁定，稍后刷新可查看最终结果。</p>
          <p v-else-if="activeOrder.orderStatus === 1">库存已锁定，后续接入支付后可继续完成支付。</p>
          <p v-else>本次抢购未成功，Redis 预扣库存会自动回补。</p>
        </div>
        <el-button class="refresh-button" type="primary" plain @click="loadOrderDetail">
          刷新结果
        </el-button>
      </section>

      <section class="order-list-panel">
        <div class="section-head">
          <h2>我的订单</h2>
          <el-button text @click="reloadOrderList">刷新</el-button>
        </div>

        <div class="order-list" v-loading="listLoading">
          <button
            v-for="order in orderList"
            :key="order.orderNo"
            class="order-card"
            type="button"
            @click="goOrderDetail(order.orderNo)"
          >
            <div class="order-cover" :style="getCoverStyle(order)">
              <span v-if="!order.coverUrl">暂无图片</span>
            </div>
            <div class="order-content">
              <div class="order-title-row">
                <h3>{{ order.productName }}</h3>
                <span :class="['order-status', orderStatusClass(order)]">
                  {{ order.orderStatusName }}
                </span>
              </div>
              <div class="order-sku">{{ order.skuName }} · 数量 {{ order.quantity }}</div>
              <div class="order-time">创建时间：{{ order.createTime }}</div>
              <div class="order-bottom">
                <span class="order-number">订单号：{{ order.orderNo }}</span>
                <span class="order-price">{{ order.totalAmountText }}</span>
              </div>
            </div>
          </button>
          <NoData v-if="!listLoading && orderList.length === 0" msg="暂无订单"></NoData>
        </div>

        <div class="load-more" v-if="canLoadMore">
          <el-button plain @click="loadMoreOrder">加载更多</el-button>
        </div>
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

const detailLoading = ref(false)
const listLoading = ref(false)
const activeOrder = ref(null)
const orderList = ref([])
const searchForm = reactive({
  keyword: '',
})
const pageInfo = reactive({
  pageNo: 1,
  pageSize: 10,
  pageTotal: 1,
})

const canLoadMore = computed(() => {
  return !listLoading.value && orderList.value.length > 0 && pageInfo.pageNo < pageInfo.pageTotal
})

onMounted(() => {
  navActionStore.setShowHeader(false)
  navActionStore.setFixedHeader(false)
  navActionStore.setFixedCategory(false)
  navActionStore.setShowCategory(false)
  navActionStore.setForceFixedHeader(false)
  loadOrderList(true)
})

const loadOrderDetail = async () => {
  const orderNo = route.params.orderNo
  if (!orderNo) {
    activeOrder.value = null
    return
  }

  detailLoading.value = true
  const result = await proxy.Request({
    url: proxy.Api.getShopOrderDetail,
    params: {
      orderNo,
    },
    showError: false,
  })
  detailLoading.value = false
  activeOrder.value = result?.data || null
}

const loadOrderList = async (isReload) => {
  if (listLoading.value) {
    return
  }
  const nextPageNo = isReload ? 1 : pageInfo.pageNo + 1
  listLoading.value = true
  const result = await proxy.Request({
    url: proxy.Api.loadShopOrder,
    params: {
      pageNo: nextPageNo,
      pageSize: pageInfo.pageSize,
    },
    showError: false,
  })
  listLoading.value = false

  const data = result?.data
  if (!data) {
    if (isReload) {
      orderList.value = []
    }
    return
  }
  const nextList = data.list || []
  orderList.value = isReload ? nextList : orderList.value.concat(nextList)
  pageInfo.pageNo = data.pageNo || nextPageNo
  pageInfo.pageTotal = data.pageTotal || 1
}

const reloadOrderList = () => {
  loadOrderList(true)
  loadOrderDetail()
}

const loadMoreOrder = () => {
  loadOrderList(false)
}

const goOrderDetail = (orderNo) => {
  if (!orderNo) {
    return
  }
  router.push(`/shop/order/${orderNo}`)
}

const goShopHome = () => {
  router.push('/shop')
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

const getCoverStyle = (order) => {
  if (!order.coverUrl) {
    return {}
  }
  return {
    backgroundImage: `url(${buildImageUrl(order.coverUrl)})`,
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

const orderStatusClass = (order) => {
  if (!order) {
    return ''
  }
  if (order.orderStatus === 0) {
    return 'locking'
  }
  if (order.orderStatus === 1) {
    return 'wait-pay'
  }
  if (order.orderStatus === 2) {
    return 'failed'
  }
  return ''
}

watch(
  () => route.params.orderNo,
  () => {
    loadOrderDetail()
  },
  { immediate: true }
)
</script>

<style lang="scss" scoped>
.shop-order-page {
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
.order-nav button {
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
  &:hover,
  &.active {
    color: #fb7299;
  }
}

.order-main {
  max-width: 1250px;
  margin: 0 auto;
  padding: 28px 0 56px;
}

.order-nav {
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

.result-panel,
.order-list-panel {
  border-radius: 6px;
  background: #fff;
  box-shadow: 0 8px 28px rgba(16, 18, 24, 0.05);
}

.result-panel {
  display: grid;
  grid-template-columns: 120px minmax(0, 1fr) 120px;
  gap: 22px;
  align-items: center;
  padding: 28px 32px;
  margin-bottom: 24px;
}

.result-status {
  height: 88px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  font-weight: 700;
  text-align: center;
  background: #909399;
  &.locking {
    background: #e6a23c;
  }
  &.wait-pay {
    background: #fb7299;
  }
  &.failed {
    background: #909399;
  }
}

.result-info {
  min-width: 0;
  h1 {
    margin: 0 0 10px;
    font-size: 24px;
    line-height: 34px;
    color: #222;
    word-break: break-word;
  }
  p {
    margin: 12px 0 0;
    color: #61666d;
  }
}

.order-no,
.result-meta,
.order-sku,
.order-time,
.order-number {
  color: #909399;
}

.result-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 18px;
}

.refresh-button {
  justify-self: end;
  border-color: #fb7299;
  color: #fb7299;
}

.order-list-panel {
  padding: 26px 32px 30px;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 18px;
  h2 {
    margin: 0;
    font-size: 22px;
    line-height: 30px;
  }
}

.order-list {
  min-height: 160px;
}

.order-card {
  width: 100%;
  min-height: 132px;
  display: grid;
  grid-template-columns: 110px minmax(0, 1fr);
  gap: 18px;
  padding: 16px 0;
  border: none;
  border-bottom: 1px solid #eef0f3;
  background: transparent;
  text-align: left;
  cursor: pointer;
  &:hover h3 {
    color: #fb7299;
  }
  &:last-child {
    border-bottom: none;
  }
}

.order-cover {
  width: 110px;
  height: 110px;
  border-radius: 6px;
  background-color: #edf0f5;
  background-size: cover;
  background-position: center;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #909399;
  overflow: hidden;
}

.order-content {
  min-width: 0;
}

.order-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  h3 {
    margin: 0;
    color: #222;
    font-size: 18px;
    line-height: 26px;
    word-break: break-word;
  }
}

.order-status {
  flex: 0 0 auto;
  padding: 3px 9px;
  border-radius: 3px;
  color: #909399;
  border: 1px solid #c8c9cc;
  &.locking {
    color: #e6a23c;
    border-color: #e6a23c;
  }
  &.wait-pay {
    color: #fb7299;
    border-color: #fb7299;
  }
}

.order-sku {
  margin-top: 10px;
}

.order-time {
  margin-top: 8px;
}

.order-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  margin-top: 16px;
}

.order-price {
  color: #fb7299;
  font-size: 20px;
  font-weight: 700;
}

.load-more {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}
</style>
