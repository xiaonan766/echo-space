package domain

import "testing"

func TestNormalizeSysSettingFillsDanmuRateDefaults(t *testing.T) {
	oldSetting := SysSetting{
		RegisterCoinCount:  10,
		PostVideoCoinCount: 5,
		VideoSize:          150,
		VideoPCount:        10,
		VideoCount:         10,
		CommentCount:       20,
		DanmuCount:         20,
	}

	got := NormalizeSysSetting(oldSetting)
	want := DefaultSysSetting()
	if got.DanmuUserRateCount != want.DanmuUserRateCount ||
		got.DanmuUserRateSeconds != want.DanmuUserRateSeconds ||
		got.DanmuUserVideoRateCount != want.DanmuUserVideoRateCount ||
		got.DanmuUserVideoRateSeconds != want.DanmuUserVideoRateSeconds ||
		got.DanmuIPRateCount != want.DanmuIPRateCount ||
		got.DanmuIPRateSeconds != want.DanmuIPRateSeconds ||
		got.DanmuVideoRateCount != want.DanmuVideoRateCount ||
		got.DanmuVideoRateSeconds != want.DanmuVideoRateSeconds {
		t.Fatalf("NormalizeSysSetting() = %+v, want danmu defaults %+v", got, want)
	}
}
