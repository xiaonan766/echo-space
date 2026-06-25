<template>
  <div class="comment-panel">
    <div class="comment-title">
      <div class="title">
        评论<span class="comment-count">{{ dataSource.totalCount }}</span>
      </div>
      <div
        :class="['order-type-item', orderType == 0 ? 'active' : '']"
        @click="changeOrder(0)"
      >
        最热
      </div>
      <el-divider direction="vertical" />
      <div
        :class="['order-type-item', orderType == 1 ? 'active' : '']"
        @click="changeOrder(1)"
      >
        最新
      </div>
    </div>
    <div class="comment-content-panel">
      <VideoCommentSend :sendType="0" :showSend="showComment"></VideoCommentSend>
      <div class="comment-list">
        <VideoCommentItem
          v-for="item in dataSource.list"
          :key="item.commentId"
          :data="item"
        ></VideoCommentItem>
        <div class="bottom-loading" v-if="loadingData">
          <img :src="proxy.Utils.getLocalImage('playing.gif')" />评论加载中...
        </div>
        <div
          class="reach-bottom"
          v-if="!loadingData && dataSource.list.length > 0 && !dataSource.hasMore"
        >
          没有更多评论
        </div>
        <NoData v-if="!loadingData && showComment && dataSource.list.length == 0"></NoData>
        <div v-if="!showComment" class="comment-closed">UP主已关闭评论区</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { mitter } from "@/eventbus/eventBus.js";
import { ACTION_TYPE } from "@/utils/Constants.js";
import VideoCommentItem from "./VideoCommentItem.vue";
import VideoCommentSend from "./VideoCommentSend.vue";
import {
  computed,
  getCurrentInstance,
  inject,
  onMounted,
  onUnmounted,
  provide,
  ref,
} from "vue";
import { useRoute } from "vue-router";

const { proxy } = getCurrentInstance();
const route = useRoute();

const videoInfo = inject("videoInfo");
const showComment = computed(() => {
  return (
    videoInfo.value.interaction == null ||
    videoInfo.value.interaction.indexOf("1") == -1
  );
});

const createEmptyCommentData = () => ({
  totalCount: 0,
  pageSize: 15,
  list: [],
  nextCursor: "",
  hasMore: false,
});

const loadingData = ref(false);
const dataSource = ref(createEmptyCommentData());
const orderType = ref(0);

const ensureCommentState = (comment) => {
  if (!comment.children) {
    comment.children = [];
  }
  if (comment.replyCount == null) {
    comment.replyCount = comment.children.length;
  }
  if (comment.replyCursor == null) {
    comment.replyCursor = "";
  }
  if (comment.replyHasMore == null) {
    comment.replyHasMore = false;
  }
  if (comment.replyLoading == null) {
    comment.replyLoading = false;
  }
  if (comment.childrenLoaded == null) {
    comment.childrenLoaded = comment.children.length > 0;
  }
  if (comment.showReply == null) {
    comment.showReply = false;
  }
  return comment;
};

const setCommentActive = (userActionMap, item) => {
  item.likeCountActive = false;
  item.hateCountActive = false;
  const userAction = userActionMap[item.commentId];
  if (!userAction) {
    return;
  }
  if (ACTION_TYPE.COMMENT_LIKE.value == userAction.actionType) {
    item.likeCountActive = true;
  } else if (ACTION_TYPE.COMMENT_HATE.value == userAction.actionType) {
    item.hateCountActive = true;
  }
};

const applyUserActions = (commentList, userActionList = []) => {
  const userActionMap = {};
  userActionList.forEach((item) => {
    userActionMap[item.commentId] = item;
  });
  commentList.forEach((item) => {
    ensureCommentState(item);
    setCommentActive(userActionMap, item);
  });
};

const resetCommentList = () => {
  dataSource.value = createEmptyCommentData();
};

const changeOrder = (_orderType) => {
  if (orderType.value == _orderType && dataSource.value.list.length > 0) {
    return;
  }
  orderType.value = _orderType;
  loadCommentList(true);
};

const loadCommentList = async (reset = false) => {
  if (!showComment.value || loadingData.value) {
    return;
  }
  if (!reset && dataSource.value.list.length > 0 && !dataSource.value.hasMore) {
    return;
  }

  loadingData.value = true;
  const result = await proxy.Request({
    url: proxy.Api.loadComment,
    params: {
      videoId: route.params.videoId,
      orderType: orderType.value,
      cursor: reset ? "" : dataSource.value.nextCursor,
    },
    showError: true,
  });
  loadingData.value = false;
  if (!result) {
    return;
  }
  if (result.data == null) {
    resetCommentList();
    return;
  }

  const commentData = result.data.commentData || createEmptyCommentData();
  const list = commentData.list || [];
  applyUserActions(list, result.data.userActionList || []);

  dataSource.value = {
    totalCount: commentData.totalCount || 0,
    pageSize: commentData.pageSize || 15,
    list: reset ? list : dataSource.value.list.concat(list),
    nextCursor: commentData.nextCursor || "",
    hasMore: !!commentData.hasMore,
  };
};

const loadReplyList = async (parentComment, reset = false) => {
  if (!parentComment || parentComment.replyLoading) {
    return;
  }
  ensureCommentState(parentComment);
  if (!reset && parentComment.childrenLoaded && !parentComment.replyHasMore) {
    return;
  }

  parentComment.replyLoading = true;
  const result = await proxy.Request({
    url: proxy.Api.loadComment,
    params: {
      videoId: route.params.videoId,
      pCommentId: parentComment.commentId,
      cursor: reset ? "" : parentComment.replyCursor,
    },
  });
  parentComment.replyLoading = false;
  if (!result || result.data == null) {
    return;
  }

  const commentData = result.data.commentData || {};
  const replyList = commentData.list || [];
  applyUserActions(replyList, result.data.userActionList || []);
  parentComment.children = reset
    ? replyList
    : (parentComment.children || []).concat(replyList);
  parentComment.replyCount = commentData.totalCount || parentComment.replyCount || 0;
  parentComment.replyCursor = commentData.nextCursor || "";
  parentComment.replyHasMore = !!commentData.hasMore;
  parentComment.childrenLoaded = true;
};

const showReplyHandler = (commentId) => {
  dataSource.value.list.forEach((item) => {
    item.showReply = commentId ? item.commentId == commentId : false;
  });
};

provide("showReply", showReplyHandler);
provide("loadReplyList", loadReplyList);

const handleWindowScroll = (curScrollTop) => {
  if (window.innerHeight + curScrollTop < document.body.offsetHeight - 30) {
    return;
  }
  loadCommentList(false);
};

const handlePostCommentSuccess = (comment) => {
  ensureCommentState(comment);
  if (comment.pCommentId === 0) {
    dataSource.value.list.unshift(comment);
    dataSource.value.totalCount++;
    return;
  }

  const parentComment = dataSource.value.list.find((item) => {
    return item.commentId == comment.pCommentId;
  });
  if (!parentComment) {
    return;
  }
  ensureCommentState(parentComment);
  parentComment.replyCount++;
  if (parentComment.childrenLoaded) {
    parentComment.children.push(comment);
  }
};

const handleDelCommentCallback = ({ pCommentId, commentId }) => {
  if (pCommentId == 0) {
    dataSource.value.list = dataSource.value.list.filter((item) => {
      return item.commentId != commentId;
    });
    dataSource.value.totalCount = Math.max(dataSource.value.totalCount - 1, 0);
    return;
  }

  const parentComment = dataSource.value.list.find((item) => {
    return item.commentId == pCommentId;
  });
  if (!parentComment) {
    return;
  }
  ensureCommentState(parentComment);
  parentComment.children = parentComment.children.filter((item) => {
    return item.commentId != commentId;
  });
  parentComment.replyCount = Math.max(parentComment.replyCount - 1, 0);
};

const handleTopCommentCallback = () => {
  loadCommentList(true);
};

onMounted(() => {
  loadCommentList(true);
  mitter.on("windowScroll", handleWindowScroll);
  mitter.on("postCommentSuccess", handlePostCommentSuccess);
  mitter.on("delCommentCallback", handleDelCommentCallback);
  mitter.on("topCommentCallback", handleTopCommentCallback);
});

onUnmounted(() => {
  mitter.off("windowScroll", handleWindowScroll);
  mitter.off("postCommentSuccess", handlePostCommentSuccess);
  mitter.off("delCommentCallback", handleDelCommentCallback);
  mitter.off("topCommentCallback", handleTopCommentCallback);
});
</script>

<style lang="scss" scoped>
.comment-panel {
  margin-top: 20px;

  .comment-title {
    display: flex;
    align-items: center;
    font-size: 15px;

    .title {
      font-size: 20px;
      font-weight: 500;

      .comment-count {
        margin-left: 5px;
        font-size: 14px;
        margin-right: 30px;
        color: var(--text2);
      }
    }

    .order-type-item {
      cursor: pointer;
    }

    .active {
      color: var(--blue);
    }
  }

  .comment-content-panel {
    padding-left: 10px;
    position: relative;

    .comment-list {
      padding-bottom: 20px;
    }
  }
}

.bottom-loading {
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text3);

  img {
    width: 20px;
    margin-right: 10px;
  }
}

.reach-bottom,
.comment-closed {
  text-align: center;
  line-height: 40px;
  color: var(--text3);
}
</style>
