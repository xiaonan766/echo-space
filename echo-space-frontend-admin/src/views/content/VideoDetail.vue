<template>
  <Dialog
    :show="dialogConfig.show"
    :title="dialogConfig.title"
    :buttons="dialogConfig.buttons"
    width="90%"
    :showCancel="false"
    @close="closeWin"
  >
    <div class="video-detail">
      <div class="video-info">
        <el-tabs v-model="activeName">
          <el-tab-pane :label="isImagePost ? '图片列表' : '视频分P'" name="video">
            <div class="video-tips" v-if="!isImagePost">红色标题代表视频有更新</div>
            <el-scrollbar :max-height="400" class="video-list">
              <div
                :class="['video-item', index == currentP - 1 ? 'active' : '']"
                v-for="(item, index) in videoFileList"
                @click="selectFile(index + 1)"
              >
                <div class="playing" v-if="!isImagePost && index == currentP - 1"></div>
                <div
                  :class="['title', item.updateType == 1 ? 'update' : '']"
                  :title="item.title"
                >
                  {{ isImagePost ? `图${index + 1}` : `P${index + 1}` }} {{ item.fileName }}
                </div>
                <div class="duration" v-if="!isImagePost">
                  {{ proxy.Utils.convertSecondsToHMS(item.duration) }}
                </div>
              </div>
            </el-scrollbar>
          </el-tab-pane>
          <el-tab-pane label="基本信息" name="base">
            <div class="video-base-info">
              <div class="base-info-item">
                <div class="item-title">标题：</div>
                <div class="item-value">{{ videoInfo.videoName }}</div>
              </div>
              <div class="base-info-item">
                <div class="item-title">发布人：</div>
                <div class="item-value">{{ videoInfo.nickName }}</div>
              </div>
              <div class="base-info-item">
                <div class="item-title">类型：</div>
                <div class="item-value">
                  {{ videoInfo.postType == 0 ? "自制" : "转载" }}
                </div>
              </div>

              <div v-if="videoInfo.postType == 1" class="base-info-item">
                <div class="item-title">资源说明：</div>
                <div class="item-value">{{ videoInfo.originInfo }}</div>
              </div>
              <div class="base-info-item">
                <div class="item-title">标签：</div>
                <div class="item-value">{{ videoInfo.tags }}</div>
              </div>
              <div class="base-info-item">
                <div class="item-title">简介：</div>
                <div class="item-value">{{ videoInfo.introduction }}</div>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
      <div class="video-play">
        <Player v-if="!isImagePost" ref="playerRef" :autoplay="false"></Player>
        <div class="image-preview" v-else>
          <Cover
            v-if="currentImage"
            :source="currentImage.filePath"
            fit="contain"
            :preview="true"
            :lazy="false"
          ></Cover>
          <NoData v-else msg="暂无图片"></NoData>
        </div>
      </div>
    </div>
  </Dialog>
</template>

<script setup>
import { mitter } from "@/eventbus/eventBus.js";
import { computed, ref, reactive, getCurrentInstance, nextTick } from "vue";
const { proxy } = getCurrentInstance();
import { useRoute, useRouter } from "vue-router";
const route = useRoute();
const router = useRouter();

const dialogConfig = ref({
  show: false,
});

const activeName = ref("video");

const videoInfo = ref();
const videoFileList = ref([]);
const isImagePost = computed(() => videoInfo.value?.contentType == 1);
const currentImage = computed(() => videoFileList.value[currentP.value - 1]);
//播放器
const playerRef = ref();
//当前播放的视频
const currentP = ref(1);
const show = (data) => {
  dialogConfig.value.show = true;
  videoInfo.value = Object.assign({}, data);
  activeName.value = "video";
  currentP.value = 1;
  loadPList();
};

const loadPList = async () => {
  let result = await proxy.Request({
    url: proxy.Api.loadVideoPList,
    params: {
      videoId: videoInfo.value.videoId,
    },
  });
  if (!result) {
    return;
  }
  videoFileList.value = result.data;
  if (isImagePost.value) {
    return;
  }
  nextTick(() => {
    playerRef.value.showPlayer(window.innerHeight - 150);
    selectVideoFile();
  });
};

const selectFile = (index) => {
  currentP.value = index;
  if (isImagePost.value) {
    return;
  }
  selectVideoFile();
};

const selectVideoFile = () => {
  if (!videoFileList.value[currentP.value - 1]) {
    return;
  }
  mitter.emit("changeP", videoFileList.value[currentP.value - 1].fileId);
};

const closeWin = () => {
  dialogConfig.value.show = false;
  if (!isImagePost.value) {
    playerRef.value?.destroyPlayer();
  }
};

defineExpose({
  show,
});
</script>

<style lang="scss" scoped>
.video-detail {
  display: flex;
  .video-info {
    width: 400px;
    .video-base-info {
      padding-right: 10px;
      .base-info-item {
        margin-top: 5px;
        display: flex;
        .item-title {
          width: 60px;
          text-align: right;
          font-size: 15px;
        }
        .item-value {
          flex: 1;
        }
      }
    }
    .video-tips {
      font-size: 13px;
      color: red;
    }
    .video-list {
      .video-item {
        padding: 6px 8px 6px 0px;
        display: flex;
        align-items: center;
        cursor: pointer;
        margin-top: 5px;
        border-radius: 3px;
        .playing {
          width: 14px;
          height: 14px;
          margin-right: 5px;
          background-position: center center;
          background-size: cover;
          background-repeat: no-repeat;
          background-image: url("@/assets/playing.gif");
        }
        .title {
          font-size: 14px;
          flex: 1;
          overflow: hidden;
          white-space: nowrap;
          text-overflow: ellipsis;
        }
        .update {
          color: red;
        }
        .duration {
          margin-left: 5px;
        }
        &:hover {
          background: #fff;
        }
      }
      .active {
        background: #fff;
      }
    }
  }
  .video-play {
    flex: 1;
    min-width: 0;
    .image-preview {
      height: calc(100vh - 150px);
      background: #f6f7f8;
      border-radius: 6px;
      overflow: hidden;
    }
  }
}
</style>
