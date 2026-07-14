<template>
  <div class="gallery-page">
    <div class="gallery-header">
      <div class="title">图库</div>
      <el-button v-if="isSearchMode" link type="primary" @click="handleClearSearch">
        清除搜索 / 返回全部图库
      </el-button>
    </div>

    <div v-if="isSearchMode" class="result-title">
      搜索结果
      <span v-if="activeSearchType === 'text' && activeKeyword">“{{ activeKeyword }}”</span>
      <span v-if="activeSearchType === 'image'">以图搜图</span>
    </div>

    <DataLoadMoreList
      :dataSource="dataSource"
      :loading="loadingData"
      :gridCount="5"
      :noDataMsg="isSearchMode ? '没有找到相似图片' : '暂无图片'"
      loadEndMsg="已经到底啦~"
      @loadData="loadCurrentPage"
    >
      <template #default="{ data }">
        <GalleryImageItem :data="data" :marginTop="20" />
      </template>
    </DataLoadMoreList>
  </div>
</template>

<script setup>
import { getCurrentInstance, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useNavAction } from '@/stores/navActionStore'
import { mitter } from '@/eventbus/eventBus.js'
import GalleryImageItem from './GalleryImageItem.vue'

const { proxy } = getCurrentInstance()
const route = useRoute()
const router = useRouter()
const navActionStore = useNavAction()

const loadingData = ref(false)
const activeSearchType = ref('')
const activeKeyword = ref('')
const searchToken = ref('')
const selectedFile = ref(null)
const isSearchMode = ref(false)
const dataSource = ref({ list: [], pageNo: 1, pageSize: 15, pageTotal: 0 })

const resetDataSource = () => {
  dataSource.value = { list: [], pageNo: 1, pageSize: 15, pageTotal: 0 }
}

const loadNormalGallery = async () => {
  const currentList = dataSource.value.list || []
  loadingData.value = true
  const result = await proxy.Request({
    url: proxy.Api.loadGalleryImageList,
    params: { pageNo: dataSource.value.pageNo, pageSize: dataSource.value.pageSize },
  })
  loadingData.value = false
  if (!result) return
  dataSource.value = Object.assign({}, result.data)
  if (result.data.pageNo > 1) dataSource.value.list = currentList.concat(result.data.list || [])
}

const loadSearchResults = async () => {
  const currentList = dataSource.value.list || []
  const params = {
    pageNo: dataSource.value.pageNo,
    pageSize: dataSource.value.pageSize,
  }
  if (searchToken.value) {
    params.searchToken = searchToken.value
  } else if (activeSearchType.value === 'text') {
    params.searchType = 'text'
    params.keyword = activeKeyword.value
  } else {
    params.searchType = 'image'
    params.file = selectedFile.value
  }
  loadingData.value = true
  const result = await proxy.Request({ url: proxy.Api.searchGallery, params, timeout: 30 * 1000 })
  loadingData.value = false
  if (!result) return
  searchToken.value = result.data.searchToken
  const list = result.data.list || []
  dataSource.value = {
    list: result.data.pageNo > 1 ? currentList.concat(list) : list,
    pageNo: result.data.pageNo,
    pageSize: result.data.pageSize,
    pageTotal: result.data.hasMore ? result.data.pageNo + 1 : result.data.pageNo,
  }
}

const loadCurrentPage = () => (isSearchMode.value ? loadSearchResults() : loadNormalGallery())

const startImageSearch = async (file) => {
  if (!file) return
  if (file.size > 10 * 1024 * 1024) {
    proxy.Message.warning('图片大小不能超过 10MB')
    return
  }
  selectedFile.value = file
  isSearchMode.value = true
  activeSearchType.value = 'image'
  activeKeyword.value = ''
  searchToken.value = ''
  await router.replace({ path: '/gallery' })
  resetDataSource()
  await loadSearchResults()
}

const startTextSearch = async (value) => {
  selectedFile.value = null
  activeSearchType.value = 'text'
  activeKeyword.value = value
  searchToken.value = ''
  isSearchMode.value = true
  resetDataSource()
  await loadSearchResults()
}

const handleClearSearch = () => router.push({ path: '/gallery' })

const clearSelectedFile = () => { selectedFile.value = null }

watch(
  () => [route.query.mode, route.query.keyword],
  async ([mode, queryKeyword]) => {
    if (mode === 'text' && typeof queryKeyword === 'string' && queryKeyword.trim()) {
      await startTextSearch(queryKeyword.trim())
      return
    }
    if (activeSearchType.value === 'image' && isSearchMode.value) return
    isSearchMode.value = false
    activeSearchType.value = ''
    activeKeyword.value = ''
    searchToken.value = ''
    resetDataSource()
    await loadNormalGallery()
  },
  { immediate: true }
)

onMounted(() => {
  navActionStore.setShowHeader(true)
  navActionStore.setFixedHeader(true)
  navActionStore.setFixedCategory(false)
  navActionStore.setShowCategory(false)
  navActionStore.setForceFixedHeader(false)
  mitter.on('galleryImageSearch', startImageSearch)
})

onUnmounted(() => {
  mitter.off('galleryImageSearch', startImageSearch)
  clearSelectedFile()
})
</script>

<style lang="scss" scoped>
.gallery-page {
  padding: 24px 0 40px;
  .gallery-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
  .title { font-size: 24px; font-weight: 600; color: var(--text); }
}
.result-title { margin-top: 22px; font-size: 18px; font-weight: 600; color: var(--text); }
.result-title span { margin-left: 8px; font-size: 14px; font-weight: 400; color: var(--text3); }
</style>
