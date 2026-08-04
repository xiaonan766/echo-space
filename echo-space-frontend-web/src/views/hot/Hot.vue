<template>
  <div class="hot-container">
    <div class="hot-part-title-panel">
      <div class="hot-24">
        <div class="iconfont icon-hot"></div>
        <div>实时热榜</div>
      </div>
    </div>
    <div class="data-list">
      <DataLoadMoreList
        :dataSource="dataSource"
        :loading="loadingData"
        @loadData="loadDataList"
        :gridCount="2"
        noDataMsg="暂无热榜视频"
      >
        <template #default="{ data }">
          <div class="hot-video-item">
            <div class="hot-rank-info">
              <span class="rank">NO.{{ data.rank || '-' }}</span>
              <span class="heat">{{ data.heatScore || 0 }} 热度</span>
            </div>
            <VideoItem :data="data" :marginTop="12" :layoutType="1"></VideoItem>
          </div>
        </template>
      </DataLoadMoreList>
    </div>
  </div>
</template>

<script setup>
import { ref, getCurrentInstance } from 'vue'

const { proxy } = getCurrentInstance()

const loadingData = ref(false)
const dataSource = ref({
  pageNo: 1,
  pageSize: 20,
  pageTotal: 0,
  list: [],
})

const loadDataList = async () => {
  const params = {
    pageNo: dataSource.value.pageNo,
    pageSize: dataSource.value.pageSize,
  }
  loadingData.value = true
  const result = await proxy.Request({
    url: proxy.Api.hotVideoList,
    params,
  })
  loadingData.value = false
  if (!result) {
    return
  }

  const currentList = dataSource.value.list || []
  dataSource.value = Object.assign(
    {
      pageNo: 1,
      pageSize: 20,
      pageTotal: 0,
      list: [],
    },
    result.data
  )
  if (result.data.pageNo > 1) {
    dataSource.value.list = currentList.concat(result.data.list || [])
  }
}

loadDataList()
</script>

<style lang="scss" scoped>
.hot-container {
  margin: 20px auto 0px;
  min-width: 1070px;
  max-width: 1286px;
  .hot-part-title-panel {
    border-bottom: 1px solid #ddd;
    padding: 10px 0px 20px 0px;
    display: flex;
    .hot-24 {
      font-size: 20px;
      display: flex;
      align-items: center;
      position: relative;
      &::after {
        content: '';
        position: absolute;
        border-bottom: 2px solid var(--blue);
        width: 100%;
        bottom: -20px;
      }
      .icon-hot {
        width: 46px;
        height: 46px;
        background: #f07775;
        color: #fff;
        font-size: 20px;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        margin-right: 10px;
      }
    }
  }
  .data-list {
    margin-top: 10px;
    .hot-video-item {
      padding: 8px 0px 12px;
      border-bottom: 1px solid #f1f2f3;
      .hot-rank-info {
        display: flex;
        align-items: center;
        justify-content: space-between;
        color: #9499a0;
        font-size: 13px;
        .rank {
          color: #f07775;
          font-weight: 600;
        }
      }
    }
  }
}
</style>
