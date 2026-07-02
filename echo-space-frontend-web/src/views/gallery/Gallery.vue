<template>
  <div class="gallery-page">
    <div class="gallery-header">
      <div class="title">图库</div>
      <div class="sub-title">浏览已审核通过的图片投稿</div>
    </div>
    <DataLoadMoreList
      :dataSource="dataSource"
      :loading="loadingData"
      :gridCount="5"
      noDataMsg="暂无图片"
      loadEndMsg="已经到底啦~~"
      @loadData="loadImageList"
    >
      <template #default="{ data }">
        <GalleryImageItem :data="data" :marginTop="20"></GalleryImageItem>
      </template>
    </DataLoadMoreList>
  </div>
</template>

<script setup>
import { getCurrentInstance, onMounted, ref } from 'vue'
import { useNavAction } from '@/stores/navActionStore'
import GalleryImageItem from './GalleryImageItem.vue'

const { proxy } = getCurrentInstance()
const navActionStore = useNavAction()

const loadingData = ref(false)
const dataSource = ref({
  list: [],
  pageNo: 1,
  pageSize: 15,
  pageTotal: 0,
})

const loadImageList = async () => {
  const currentList = dataSource.value.list || []
  loadingData.value = true
  const result = await proxy.Request({
    url: proxy.Api.loadGalleryImageList,
    params: {
      pageNo: dataSource.value.pageNo,
      pageSize: dataSource.value.pageSize,
    },
  })
  loadingData.value = false
  if (!result) {
    return
  }
  dataSource.value = Object.assign({}, result.data)
  if (result.data.pageNo > 1) {
    dataSource.value.list = currentList.concat(result.data.list || [])
  }
}

onMounted(() => {
  navActionStore.setShowHeader(true)
  navActionStore.setFixedHeader(true)
  navActionStore.setFixedCategory(false)
  navActionStore.setShowCategory(false)
  navActionStore.setForceFixedHeader(false)
  loadImageList()
})
</script>

<style lang="scss" scoped>
.gallery-page {
  padding: 24px 0 40px;
  .gallery-header {
    display: flex;
    align-items: baseline;
    margin-bottom: 4px;
    .title {
      font-size: 24px;
      font-weight: 600;
      color: var(--text);
    }
    .sub-title {
      margin-left: 14px;
      color: #9499a0;
      font-size: 14px;
    }
  }
}
</style>
