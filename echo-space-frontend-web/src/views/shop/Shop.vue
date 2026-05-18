<template>
  <div class="shop-page">
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

    <div class="shop-tabs">
      <button
        v-for="item in shopTypes"
        :key="item.type"
        :class="['tab-item', activeType === item.type ? 'active' : '']"
        type="button"
        @click="handleTypeChange(item.type)"
      >
        {{ item.label }}
      </button>
    </div>

    <section class="recommend-section" v-loading="recommendLoading">
      <div class="section-title">
        <span>{{ recommendTitle }}</span>
      </div>
      <div v-if="recommendList.length > 0" class="recommend-list">
        <button
          v-for="item in recommendList"
          :key="getItemKey(item)"
          class="recommend-card"
          type="button"
          :style="getImageStyle(item)"
          @click="openShopDetail(item)"
        >
          <span v-if="!getItemImage(item)" class="recommend-empty-image">暂无图片</span>
          <div class="banner-mask">
            <div class="banner-label">系统推荐</div>
            <div class="banner-name">{{ getItemName(item) }}</div>
          </div>
        </button>
      </div>
      <div v-else class="empty-banner">
        {{ currentTypeLabel }}推荐内容待接入
      </div>
    </section>

    <section class="list-section" v-loading="listLoading">
      <div class="section-title">
        <span>{{ listTitle }}</span>
      </div>
      <div class="shop-list" v-if="shopItemList.length > 0">
        <article class="shop-card" v-for="item in shopItemList" :key="getItemKey(item)">
          <button
            class="item-cover"
            type="button"
            :style="getImageStyle(item)"
            @click="openShopDetail(item)"
          >
            <span v-if="!getItemImage(item)">暂无图片</span>
          </button>
          <div class="item-info">
            <button class="item-name" type="button" @click="openShopDetail(item)">
              {{ getItemName(item) }}
            </button>
            <div class="item-meta" v-if="getItemTime(item)">
              <span class="iconfont icon-content"></span>
              <span>{{ getItemTime(item) }}</span>
            </div>
            <div class="item-meta" v-if="getItemAddress(item)">
              <span class="iconfont icon-home"></span>
              <span>{{ getItemAddress(item) }}</span>
            </div>
            <div class="item-meta" v-if="getItemStock(item)">
              <span class="iconfont icon-list"></span>
              <span>{{ getItemStock(item) }}</span>
            </div>
            <div class="item-bottom">
              <div :class="['item-price', hasPrice(item) ? '' : 'price-muted']">
                {{ getItemPrice(item) }}
              </div>
              <div class="item-status" v-if="getItemStatus(item)">
                {{ getItemStatus(item) }}
              </div>
            </div>
          </div>
        </article>
      </div>
      <NoData v-else :msg="emptyText"></NoData>
      <div class="load-more" v-if="hasMore">
        <el-button @click="loadMoreList">加载更多</el-button>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, getCurrentInstance, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useLoginStore } from '@/stores/loginStore.js'
import { useNavAction } from '@/stores/navActionStore'

const { proxy } = getCurrentInstance()
const router = useRouter()
const route = useRoute()
const loginStore = useLoginStore()
const navActionStore = useNavAction()

const shopTypes = [
  {
    label: '演出',
    type: 'show',
  },
  {
    label: '周边',
    type: 'goods',
  },
]

const activeType = ref(route.query.type === 'goods' ? 'goods' : 'show')
const searchForm = reactive({
  keyword: route.query.keyword || '',
})
const recommendLoading = ref(false)
const listLoading = ref(false)
const recommendList = ref([])
const shopItemList = ref([])
const pageInfo = reactive({
  pageNo: 1,
  pageSize: 8,
  pageTotal: 1,
})

const currentTypeLabel = computed(() => {
  const currentType = shopTypes.find((item) => item.type === activeType.value)
  return currentType ? currentType.label : '演出'
})
const recommendTitle = computed(() => `热门${currentTypeLabel.value}`)
const listTitle = computed(() => `${currentTypeLabel.value}列表`)
const emptyText = computed(() => `暂无${currentTypeLabel.value}数据`)
const hasMore = computed(() => pageInfo.pageNo < pageInfo.pageTotal)

onMounted(() => {
  navActionStore.setShowHeader(false)
  navActionStore.setFixedHeader(false)
  navActionStore.setFixedCategory(false)
  navActionStore.setShowCategory(false)
  navActionStore.setForceFixedHeader(false)
})

const normalizeList = (data) => {
  if (Array.isArray(data)) {
    return data
  }
  return data?.list || data?.records || data?.rows || []
}

const loadRecommendList = async () => {
  recommendLoading.value = true
  const result = await proxy.Request({
    url: proxy.Api.loadShopRecommend,
    params: {
      itemType: activeType.value,
    },
    showError: false,
  })
  recommendLoading.value = false
  recommendList.value = result ? normalizeList(result.data) : []
}

const loadShopItemList = async (isReload = true) => {
  const pageNo = isReload ? 1 : pageInfo.pageNo + 1
  listLoading.value = true
  const result = await proxy.Request({
    url: proxy.Api.loadShopList,
    params: {
      itemType: activeType.value,
      keyword: searchForm.keyword,
      pageNo,
      pageSize: pageInfo.pageSize,
    },
    showError: false,
  })
  listLoading.value = false
  if (!result) {
    if (isReload) {
      shopItemList.value = []
      pageInfo.pageNo = 1
      pageInfo.pageTotal = 1
    }
    return
  }

  const data = result.data || {}
  const nextList = normalizeList(data)
  shopItemList.value = isReload ? nextList : shopItemList.value.concat(nextList)
  pageInfo.pageNo = data.pageNo || pageNo
  pageInfo.pageTotal = data.pageTotal || data.pageCount || data.totalPage || 1
}

const reloadShopData = () => {
  loadRecommendList()
  loadShopItemList(true)
}

const handleTypeChange = (type) => {
  if (activeType.value === type) {
    return
  }
  activeType.value = type
  router.replace({
    path: '/shop',
    query: {
      type,
      keyword: searchForm.keyword || undefined,
    },
  })
}

const handleSearch = () => {
  const nextKeyword = searchForm.keyword || ''
  if (
    route.query.type === activeType.value &&
    (route.query.keyword || '') === nextKeyword
  ) {
    loadShopItemList(true)
    return
  }
  router.replace({
    path: '/shop',
    query: {
      type: activeType.value,
      keyword: nextKeyword || undefined,
    },
  })
}

const loadMoreList = () => {
  loadShopItemList(false)
}

const goShopHome = () => {
  activeType.value = 'show'
  searchForm.keyword = ''
  if (route.path === '/shop' && Object.keys(route.query).length === 0) {
    reloadShopData()
    return
  }
  router.replace('/shop')
}

const goMainSite = () => {
  router.push('/')
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

const getItemKey = (item) => {
  return (
    item.itemId ||
    item.goodsId ||
    item.eventId ||
    item.id ||
    `${activeType.value}-${getItemName(item)}`
  )
}

const getItemName = (item) => {
  return (
    item.itemName ||
    item.goodsName ||
    item.eventName ||
    item.name ||
    item.title ||
    '未命名商品'
  )
}

const getItemImage = (item) => {
  return item.image || item.imageUrl || item.cover || item.coverUrl || item.poster || item.posterUrl
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

const getImageStyle = (item) => {
  const image = getItemImage(item)
  if (!image) {
    return {}
  }
  return {
    backgroundImage: `url(${buildImageUrl(image)})`,
  }
}

const getItemTime = (item) => {
  if (item.saleStartText) {
    return item.saleStartText
  }
  if (item.timeText) {
    return item.timeText
  }
  if (item.dateText) {
    return item.dateText
  }
  if (item.startTime && item.endTime) {
    return `${item.startTime} - ${item.endTime}`
  }
  return item.startTime || item.showTime || ''
}

const getItemAddress = (item) => {
  return item.address || item.venue || item.placeName || item.city || ''
}

const getItemStock = (item) => {
  if (activeType.value !== 'goods') {
    return ''
  }
  if (item.stockText) {
    return item.stockText
  }
  if (item.availableStock === 0) {
    return '暂无库存'
  }
  return item.availableStock ? `库存 ${item.availableStock}` : ''
}

const hasPrice = (item) => {
  return item.priceText || item.price || item.minPrice || item.salePrice
}

const getItemPrice = (item) => {
  if (item.priceText) {
    return item.priceText
  }
  const price = item.minPrice || item.salePrice || item.price
  if (!price) {
    return '价格待定'
  }
  const priceText = `￥${Number(price).toFixed(2)}`
  return activeType.value === 'goods' ? priceText : `${priceText} 起`
}

const getItemStatus = (item) => {
  return item.saleStatusName || item.statusName || item.statusText || ''
}

const getDetailPath = (item) => {
  return item.detailUrl || item.detailPath || ''
}

const openShopDetail = (item) => {
  const detailPath = getDetailPath(item)
  if (!detailPath) {
    proxy.Message.warning('购买页面待接入')
    return
  }
  if (/^(https?:)?\/\//.test(detailPath)) {
    window.open(detailPath, '_blank')
    return
  }
  router.push(detailPath)
}

watch(
  () => route.query,
  () => {
    activeType.value = route.query.type === 'goods' ? 'goods' : 'show'
    searchForm.keyword = route.query.keyword || ''
    reloadShopData()
  },
  { immediate: true, deep: true }
)
</script>

<style lang="scss" scoped>
.shop-page {
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
.tab-item,
.item-cover,
.item-name,
.recommend-card {
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

.shop-tabs {
  height: 86px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 58px;
  background: #fff;
  .tab-item {
    position: relative;
    font-size: 20px;
    line-height: 42px;
    color: #222;
    &:hover {
      color: #fb7299;
    }
  }
  .active {
    color: #fb7299;
    font-weight: 700;
    &::after {
      content: '';
      position: absolute;
      left: 50%;
      bottom: 0;
      width: 32px;
      height: 3px;
      border-radius: 3px;
      background: #fb7299;
      transform: translateX(-50%);
    }
  }
}

.recommend-section,
.list-section {
  max-width: 1250px;
  margin: 0 auto;
}

.section-title {
  padding: 28px 0 16px;
  font-size: 22px;
  font-weight: 700;
}

.recommend-list,
.empty-banner {
  width: 100%;
  min-height: 280px;
  border-radius: 6px;
}

.recommend-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 18px;
}

.recommend-card {
  min-height: 280px;
  overflow: hidden;
  background-color: #e7e9ee;
  background-size: cover;
  background-position: center;
  text-align: left;
  position: relative;
  .banner-mask {
    position: absolute;
    inset: 0;
    padding: 36px;
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
    background: linear-gradient(180deg, rgba(0, 0, 0, 0.08), rgba(0, 0, 0, 0.58));
    color: #fff;
  }
  .banner-label {
    font-size: 14px;
    margin-bottom: 8px;
  }
  .banner-name {
    max-width: 720px;
    font-size: 24px;
    font-weight: 700;
    line-height: 1.3;
  }
}

.recommend-empty-image {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text3);
}

.empty-banner {
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
  color: var(--text3);
  border: 1px dashed #d9dce1;
}

.shop-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 34px 40px;
  padding-bottom: 40px;
}

.shop-card {
  height: 170px;
  display: grid;
  grid-template-columns: 128px minmax(0, 1fr);
  column-gap: 22px;
  padding: 18px;
  border-radius: 6px;
  background: #fff;
  box-shadow: 0 8px 24px rgba(16, 18, 24, 0.04);
}

.item-cover {
  width: 128px;
  height: 134px;
  border-radius: 6px;
  overflow: hidden;
  background-color: #edf0f5;
  background-size: cover;
  background-position: center;
  color: var(--text3);
  display: flex;
  align-items: center;
  justify-content: center;
}

.item-info {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.item-name {
  color: var(--text);
  font-size: 18px;
  line-height: 26px;
  text-align: left;
  display: -webkit-box;
  overflow: hidden;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  &:hover {
    color: #fb7299;
  }
}

.item-meta {
  display: flex;
  align-items: center;
  margin-top: 10px;
  color: var(--text3);
  line-height: 20px;
  min-width: 0;
  span:last-child {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .iconfont {
    flex: 0 0 auto;
    margin-right: 8px;
    font-size: 14px;
  }
}

.item-bottom {
  margin-top: auto;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
}

.item-price {
  color: #fb7299;
  font-size: 24px;
  line-height: 30px;
}

.price-muted {
  color: var(--text3);
  font-size: 16px;
}

.item-status {
  color: #fb7299;
  border: 1px solid #fb7299;
  border-radius: 3px;
  padding: 2px 6px;
  line-height: 18px;
  white-space: nowrap;
}

.load-more {
  padding: 0 0 40px;
  text-align: center;
}
</style>
