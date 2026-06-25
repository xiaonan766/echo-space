<template>
  <div class="comment-item">
    <Avatar
      :width="replyLevel == 1 ? 50 : 30"
      :avatar="data.avatar"
      :userId="data.userId"
    ></Avatar>
    <div class="comment-content-panel">
      <div class="nick-name-panel">
        <router-link :to="`/user/${data.userId}`" class="nick-name">
          {{ data.nickName }}
        </router-link>
        <template v-if="data.replyUserId">
          <div class="reply-title">回复</div>
          <router-link :to="`/user/${data.replyUserId}`" class="reply-nick-name">
            @{{ data.replyNickName }}
          </router-link>
        </template>
      </div>
      <div class="comment-message">
        <Tag :type="0" v-if="data.topType == 1"></Tag>
        <span v-html="proxy.Utils.resetHtmlContent(data.content)"></span>
      </div>
      <div v-if="data.imgPath" class="image-show">
        <Cover
          :source="data.imgPath + proxy.imageThumbnailSuffix"
          :preview="true"
          fit="cover"
        ></Cover>
      </div>
      <div class="comment-op">
        <div class="op-left">
          <div class="comment-time">{{ data.postTime }}</div>
          <div
            :class="['iconfont icon-good', data.likeCountActive ? 'active' : '']"
            @click="doLike(data)"
          >
            {{ data.likeCount == 0 ? "" : data.likeCount }}
          </div>
          <div
            :class="['iconfont icon-no-good', data.hateCountActive ? 'active' : '']"
            @click="doHate(data)"
          >
            {{ data.hateCount == 0 ? "" : data.hateCount }}
          </div>
          <div class="reply-btn" @click="showReplyHandler(data, replyLevel)">
            回复
          </div>
        </div>
        <el-dropdown v-if="canOperateComment">
          <span class="op-right iconfont icon-more"> </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                @click="topComment"
                v-if="currentVideoUserId == loginUserId && data.pCommentId == 0"
              >
                {{ data.topType == 1 ? "取消置顶" : "置顶" }}
              </el-dropdown-item>
              <el-dropdown-item @click="delComment">删除</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
      <div class="reply-list" v-if="replyLevel == 1">
        <VideoCommentItem
          v-for="item in data.children"
          :key="item.commentId"
          :data="item"
          :replyLevel="2"
        />
        <div class="reply-more-panel">
          <span
            v-if="data.replyLoading"
            class="reply-more-text reply-loading"
          >
            回复加载中...
          </span>
          <span
            v-else-if="data.replyCount > 0 && !data.childrenLoaded"
            class="reply-more-text"
            @click="loadReplies(true)"
          >
            展开 {{ data.replyCount }} 条回复
          </span>
          <span
            v-else-if="data.childrenLoaded && data.replyHasMore"
            class="reply-more-text"
            @click="loadReplies(false)"
          >
            加载更多回复
          </span>
        </div>
      </div>
      <VideoCommentSend
        v-if="replyLevel == 1 && data.showReply"
        :sendType="1"
      ></VideoCommentSend>
    </div>
  </div>
</template>

<script setup>
import Tag from "@/components/Tag.vue";
import { doUserAction } from "@/utils/Api";
import { ACTION_TYPE } from "@/utils/Constants.js";
import VideoCommentSend from "./VideoCommentSend.vue";
import Avatar from "@/components/Avatar.vue";
import { computed, getCurrentInstance, inject, nextTick } from "vue";
import { useRoute } from "vue-router";
import { useLoginStore } from "@/stores/loginStore";
import { mitter } from "@/eventbus/eventBus.js";

const { proxy } = getCurrentInstance();
const route = useRoute();
const loginStore = useLoginStore();

const props = defineProps({
  data: {
    type: Object,
    default: () => ({}),
  },
  replyLevel: {
    type: Number,
    default: 1,
  },
});

const videoInfo = inject("videoInfo");
const showReply = inject("showReply");
const loadReplyList = inject("loadReplyList");

const currentVideoInfo = computed(() => {
  return videoInfo?.value || videoInfo || {};
});
const currentVideoUserId = computed(() => currentVideoInfo.value.userId || "");
const loginUserId = computed(() => loginStore.userInfo?.userId || "");
const canOperateComment = computed(() => {
  return (
    loginUserId.value &&
    (props.data.userId == loginUserId.value ||
      currentVideoUserId.value == loginUserId.value)
  );
});

const showReplyHandler = (item, replyLevel) => {
  if (showReply) {
    showReply(replyLevel == 1 ? item.commentId : item.pCommentId);
  }
  nextTick(() => {
    const commentData = {
      replyCommentId: item.commentId,
      nickName: item.nickName,
    };
    mitter.emit("initCommentData", commentData);
  });
};

const loadReplies = async (reset = false) => {
  if (!loadReplyList) {
    return;
  }
  await loadReplyList(props.data, reset);
};

const doLike = (data) => {
  doUserAction(
    {
      videoId: route.params.videoId,
      actionType: ACTION_TYPE.COMMENT_LIKE.value,
      commentId: data.commentId,
    },
    () => {
      if (data.hateCountActive) {
        data.hateCountActive = false;
        data.hateCount = Math.max(data.hateCount - 1, 0);
      }
      if (data.likeCountActive) {
        data.likeCountActive = false;
        data.likeCount = Math.max(data.likeCount - 1, 0);
      } else {
        data.likeCount++;
        data.likeCountActive = true;
      }
    }
  );
};

const doHate = (data) => {
  doUserAction(
    {
      videoId: route.params.videoId,
      actionType: ACTION_TYPE.COMMENT_HATE.value,
      commentId: data.commentId,
    },
    () => {
      if (data.likeCountActive) {
        data.likeCountActive = false;
        data.likeCount = Math.max(data.likeCount - 1, 0);
      }
      if (data.hateCountActive) {
        data.hateCountActive = false;
        data.hateCount = Math.max(data.hateCount - 1, 0);
      } else {
        data.hateCount++;
        data.hateCountActive = true;
      }
    }
  );
};

const delComment = () => {
  proxy.Confirm({
    message: "确定要删除评论吗？",
    okfun: async () => {
      const result = await proxy.Request({
        url: proxy.Api.userDelComment,
        params: {
          commentId: props.data.commentId,
        },
      });
      if (!result) {
        return;
      }
      mitter.emit("delCommentCallback", {
        pCommentId: props.data.pCommentId,
        commentId: props.data.commentId,
      });
    },
  });
};

const topComment = () => {
  proxy.Confirm({
    message: `确定要${props.data.topType == 1 ? "取消置顶" : "置顶"}吗？`,
    okfun: async () => {
      const result = await proxy.Request({
        url:
          props.data.topType == 1
            ? proxy.Api.userCancelTopComment
            : proxy.Api.userTopComment,
        params: {
          commentId: props.data.commentId,
        },
      });
      if (!result) {
        return;
      }
      mitter.emit("topCommentCallback");
    },
  });
};
</script>

<style lang="scss" scoped>
.comment-item {
  display: flex;
  border-bottom: 1px solid #ddd;
  padding: 10px 0px;

  .comment-content-panel {
    flex: 1;
    margin-left: 15px;
    margin-top: 5px;

    .nick-name-panel {
      font-size: 14px;
      display: flex;
      align-items: center;
      vertical-align: middle;

      .nick-name {
        text-decoration: none;
        color: var(--text2);
      }

      .reply-title {
        margin: 0px 3px;
      }

      .reply-nick-name {
        text-decoration: none;
        color: var(--blue3);
      }
    }

    .comment-message {
      font-size: 14px;
      line-height: 20px;
      margin-top: 5px;
    }

    .image-show {
      margin-top: 10px;
      width: 100px;
      height: 100px;
    }

    .comment-op {
      margin-top: 10px;
      display: flex;
      color: var(--text3);
      align-items: center;
      font-size: 13px;
      justify-content: space-between;

      .op-left {
        display: flex;

        .comment-time {
          margin-right: 20px;
        }

        .iconfont {
          font-size: 13px;
          margin-right: 20px;
          cursor: pointer;

          &::before {
            font-size: 15px;
            margin-right: 5px;
          }
        }

        .active {
          &::before {
            color: var(--blue2);
          }
        }

        .reply-btn {
          cursor: pointer;
        }
      }

      .op-right {
        cursor: pointer;
        margin-right: 5px;
      }
    }
  }
}

.reply-list {
  .comment-item {
    border-bottom: none;
  }

  .comment-content-panel {
    .nick-name-panel {
      float: left;
      margin-right: 5px;
    }

    .comment-message {
      margin-top: 0px;
    }
  }

  .reply-more-panel {
    line-height: 28px;
    font-size: 13px;
    color: var(--text3);

    .reply-more-text {
      cursor: pointer;
      color: var(--blue);
    }

    .reply-loading {
      cursor: default;
      color: var(--text3);
    }
  }
}
</style>
