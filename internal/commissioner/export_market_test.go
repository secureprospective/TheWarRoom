package commissioner

func CloneClosedNativeForTest(raw RawClosedBooks) (NativeAssets, MarketBook, error) {
	return cloneClosedNative(raw)
}
