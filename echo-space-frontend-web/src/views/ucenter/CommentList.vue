<template>
  <div class="comment-panel">
    <VideoSearchSelect @loadData="loadData4VideoSelect"></VideoSearchSelect>
    <Table
      ref="tableInfoRef"
      :columns="columns"
      :fetch="loadDataList"
      :dataSource="tableData"
      :options="tableOptions"
      :extHeight="tableOptions.extHeight"
      :showPagination="false"
    >
      <template #slotComment="{ row }">
        <div class="comment-info">
          <Avatar :avatar="row.avatar" :userId="row.userId"></Avatar>
          <div class="comment">
            <div>
              <router-link
                target="_blank"
                class="a-link nick-name"
                :to="`/user/${row.userId}`"
                >{{ row.nickName }}
              </router-link>
              <template v-if="row.replyUserId">
                回复@
                <router-link
                  target="_blank"
                  class="a-link nick-name"
                  :to="`/user/${row.replyUserId}`"
                  >{{ row.replyNickName }} </router-link
                >的评论
              </template>
            </div>

            <div class="content">{{ row.content }}</div>
            <div v-if="row.imgPath" class="image-show">
              <Cover
                :source="row.imgPath + proxy.imageThumbnailSuffix"
                :preview="true"
                fit="cover"
              ></Cover>
            </div>
            <div class="time-info">
              <div class="time">{{ row.postTime }}</div>
              <div
                class="iconfont icon-delete"
                @click="delComment(row.commentId)"
              ></div>
            </div>
          </div>
        </div>
      </template>

      <template #slotVideo="{ row }">
        <router-link
          :to="`/video/${row.videoId}`"
          target="_blank"
          class="a-link"
        >
          <Cover :source="row.videoCover"></Cover>
          <div class="video-name">{{ row.videoName }}</div>
        </router-link>
      </template>
    </Table>

    <div class="cursor-pagination">
      <span class="page-indicator">第 {{ cursorPager.pageNo }} 页</span>
      <el-select
        v-model="cursorPager.pageSize"
        class="page-size-select"
        :disabled="isLoading"
        @change="handlePageSizeChange"
      >
        <el-option label="15条/页" :value="15"></el-option>
        <el-option label="30条/页" :value="30"></el-option>
        <el-option label="50条/页" :value="50"></el-option>
        <el-option label="100条/页" :value="100"></el-option>
      </el-select>
      <el-button
        :disabled="isLoading || !tableData.hasPrev"
        @click="handlePrevPage"
        >上一页</el-button
      >
      <el-button
        type="primary"
        :disabled="isLoading || !tableData.hasNext"
        @click="handleNextPage"
        >下一页</el-button
      >
    </div>
  </div>
</template>

<script setup>
import VideoSearchSelect from "./VideoSerchSelect.vue";
import Table from "@/components/Table.vue";
import { ref, reactive, getCurrentInstance } from "vue";
import { useRoute } from "vue-router";

const { proxy } = getCurrentInstance();
const route = useRoute();

const currentVideoId = ref(route.query.videoId || "");
const isLoading = ref(false);
const tableInfoRef = ref();
const tableOptions = ref({
  extHeight: 10,
});
const cursorPager = reactive({
  pageNo: 1,
  pageSize: 15,
  cursor: "",
});
const tableData = ref({
  list: [],
  pageSize: 15,
  nextCursor: "",
  prevCursor: "",
  hasNext: false,
  hasPrev: false,
});

const loadData4VideoSelect = (videoId) => {
  currentVideoId.value = videoId || "";
  resetCursorAndLoad();
};

const columns = [
  {
    label: "评论信息",
    scopedSlots: "slotComment",
  },
  {
    label: "视频信息",
    scopedSlots: "slotVideo",
    width: 150,
  },
];

const loadDataList = async (cursor = cursorPager.cursor) => {
  if (isLoading.value) {
    return false;
  }
  isLoading.value = true;
  let result = await proxy.Request({
    url: proxy.Api.ucLoadComment,
    params: {
      pageSize: cursorPager.pageSize,
      cursor,
      videoId: currentVideoId.value,
    },
  });
  isLoading.value = false;
  if (!result) {
    return false;
  }

  const resultData = result.data || {};
  Object.assign(tableData.value, {
    list: resultData.list || [],
    pageSize: resultData.pageSize || cursorPager.pageSize,
    nextCursor: resultData.nextCursor || "",
    prevCursor: resultData.prevCursor || "",
    hasNext: !!resultData.hasNext,
    hasPrev: !!resultData.hasPrev,
  });
  cursorPager.cursor = cursor;
  cursorPager.pageSize = tableData.value.pageSize;
  return true;
};

const resetCursorAndLoad = () => {
  cursorPager.pageNo = 1;
  cursorPager.cursor = "";
  loadDataList("");
};

const handlePageSizeChange = () => {
  resetCursorAndLoad();
};

const handleNextPage = async () => {
  if (!tableData.value.hasNext || !tableData.value.nextCursor) {
    return;
  }
  cursorPager.pageNo += 1;
  const loaded = await loadDataList(tableData.value.nextCursor);
  if (!loaded) {
    cursorPager.pageNo -= 1;
  }
};

const handlePrevPage = async () => {
  if (!tableData.value.hasPrev || !tableData.value.prevCursor) {
    return;
  }
  const previousPageNo = cursorPager.pageNo;
  cursorPager.pageNo = Math.max(1, cursorPager.pageNo - 1);
  const loaded = await loadDataList(tableData.value.prevCursor);
  if (!loaded) {
    cursorPager.pageNo = previousPageNo;
  }
};

const refreshCurrentPageAfterDelete = async () => {
  const currentCursor = cursorPager.cursor;
  const previousCursor = tableData.value.prevCursor;
  const currentPageNo = cursorPager.pageNo;
  const loaded = await loadDataList(currentCursor);
  if (
    loaded &&
    tableData.value.list.length === 0 &&
    currentPageNo > 1 &&
    previousCursor
  ) {
    cursorPager.pageNo = currentPageNo - 1;
    await loadDataList(previousCursor);
  }
};

const delComment = (commentId) => {
  proxy.Confirm({
    message: "确定要删除吗？",
    okfun: async () => {
      let result = await proxy.Request({
        url: proxy.Api.ucDelComment,
        params: {
          commentId,
        },
      });
      if (!result) {
        return;
      }
      proxy.Message.success("删除成功");
      refreshCurrentPageAfterDelete();
    },
  });
};
</script>

<style lang="scss" scoped>
.comment-panel {
  .comment-info {
    display: flex;
    .comment {
      margin-left: 10px;
    }
    .time-info {
      display: flex;
      font-size: 12px;
      .iconfont {
        margin-left: 5px;
        font-size: 13px;
        cursor: pointer;
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
  }
}
.image-show {
  width: 100px;
  height: 100px;
  overflow: hidden;
  margin: 5px 0px;
}
.cursor-pagination {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 10px;
  padding-top: 10px;
}
.page-indicator {
  color: var(--text3);
  font-size: 13px;
}
.page-size-select {
  width: 120px;
}
</style>
