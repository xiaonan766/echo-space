<template>
  <div class="top-panel">
    <el-card>
      <el-form :model="searchForm" @submit.prevent>
        <el-row :gutter="10">
          <el-col :span="5">
            <el-form-item label="视频">
              <el-input
                v-model="searchForm.videoNameFuzzy"
                clearable
                placeholder="输入视频名称搜索"
                @keyup.enter="handleSearch"
              ></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="5">
            <el-button type="success" @click="handleSearch">搜索</el-button>
          </el-col>
        </el-row>
      </el-form>
    </el-card>
  </div>
  <el-card class="table-data-card">
    <Table
      ref="tableInfoRef"
      :columns="columns"
      :fetch="loadDataList"
      :dataSource="tableData"
      :options="tableOptions"
      :extHeight="tableOptions.extHeight"
    >
      <template #slotComment="{ row }">
        <div class="comment-info">
          <a class="a-link nick-name" :href="`${proxy.webDomain}/user/${row.userId}`" target="_blank">
            <Avatar :avatar="row.avatar"></Avatar>
          </a>
          <div class="comment">
            <div>
              <a class="a-link nick-name" :href="`${proxy.webDomain}/user/${row.userId}`" target="_blank">
                {{ row.nickName || '未知用户' }}
              </a>
              <template v-if="row.replyUserId">
                回复@
                <a class="a-link nick-name" :href="`${proxy.webDomain}/user/${row.replyUserId}`" target="_blank">
                  {{ row.replyNickName || '未知用户' }}
                </a>
                的评论
              </template>
            </div>

            <div class="content">{{ row.content }}</div>
            <div v-if="row.imgPath" class="image-show">
              <Cover :source="row.imgPath + proxy.imageThumbnailSuffix" :preview="true" fit="cover"></Cover>
            </div>
            <div class="time-info">
              <div class="time">{{ row.postTime }}</div>
              <div class="iconfont icon-delete" title="删除评论" @click="delComment(row.commentId)"></div>
            </div>
          </div>
        </div>
      </template>

      <template #slotVideo="{ row }">
        <a :href="`${proxy.webDomain}/video/${row.videoId}`" target="_blank" class="a-link">
          <Cover :source="row.videoCover"></Cover>
          <div class="video-name">{{ row.videoName || '视频不存在' }}</div>
        </a>
      </template>
    </Table>
  </el-card>
</template>

<script setup>
import Table from '@/components/Table.vue'
import { ref, getCurrentInstance } from 'vue'

const { proxy } = getCurrentInstance()

const columns = [
  {
    label: '评论信息',
    scopedSlots: 'slotComment',
  },
  {
    label: '视频信息',
    scopedSlots: 'slotVideo',
    width: 150,
  },
]

const tableInfoRef = ref()
const tableOptions = ref({
  extHeight: 0,
})

const searchForm = ref({})

const tableData = ref({})
const loadDataList = async () => {
  let params = {
    pageNo: tableData.value.pageNo,
    pageSize: tableData.value.pageSize,
  }
  Object.assign(params, searchForm.value)
  let result = await proxy.Request({
    url: proxy.Api.loadComment,
    params: params,
  })
  if (!result) {
    return
  }
  Object.assign(tableData.value, result.data)
}

const handleSearch = () => {
  tableData.value.pageNo = 1
  loadDataList()
}

const delComment = (commentId) => {
  proxy.Confirm({
    message: '确定要删除这条评论吗？',
    okfun: async () => {
      let result = await proxy.Request({
        url: proxy.Api.delComment,
        params: {
          commentId,
        },
      })
      if (!result) {
        return
      }
      proxy.Message.success('删除成功')
      loadDataList()
    },
  })
}
</script>

<style lang="scss" scoped>
.comment-info {
  display: flex;
  .comment {
    margin-left: 10px;
  }
  .content {
    margin: 5px 0;
    color: var(--text2);
    line-height: 20px;
    word-break: break-all;
  }
  .time-info {
    display: flex;
    font-size: 12px;
    color: var(--text3);
    .iconfont {
      margin-left: 5px;
      font-size: 13px;
      cursor: pointer;
      &:hover {
        color: #f56c6c;
      }
    }
  }
}
.video-name {
  text-decoration: none;
  color: var(--text3);
  font-size: 13px;
  margin-top: 5px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  width: 100%;
}
.image-show {
  width: 100px;
  height: 100px;
  overflow: hidden;
  margin: 5px 0;
}
</style>
