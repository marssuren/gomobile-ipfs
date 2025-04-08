# 文件概览：Makefile
# 这是gomobile-ipfs项目的主构建文件，定义了编译、测试、发布的完整流程。
# 主要功能包括：
# 1. 为Android和iOS平台构建Go核心库
# 2. 创建移动平台桥接库
# 3. 构建示例应用
# 4. 测试框架各组件
# 5. 生成文档
# 6. 发布到Maven和CocoaPods仓库
# 
# 主要构建目标详解：
## 这两行确定了Makefile的位置和配置文件位置。$(shell ...)执行shell命令，$(MAKEFILE_LIST)是make内置变量，包含所有包含的makefile文件名。
# 获取Makefile所在的目录路径
MAKEFILE_DIR = $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
# 项目配置文件路径
MANIFEST_FILE = $(MAKEFILE_DIR)/Manifest.yml

# 工具目录和脚本定义
UTILS_DIR = $(MAKEFILE_DIR)/utils
UTIL_MANIFEST_GET_BIN = $(UTILS_DIR)/manifest_get/manifest_get.sh
UTIL_MANIFEST_GET = $(UTILS_DIR)/manifest_get
UTIL_MAVEN_FORMAT = $(UTILS_DIR)/maven_format
UTIL_MAVEN_FORMAT_REQ = $(UTIL_MAVEN_FORMAT)/requirements.txt
UTIL_MAVEN_FORMAT_CORE_BIN = $(UTIL_MAVEN_FORMAT)/maven_format_core.py
UTIL_MAVEN_PUBLISH = $(UTILS_DIR)/maven_publish
UTIL_MAVEN_PUBLISH_REQ = $(UTIL_MAVEN_PUBLISH)/requirements.txt
UTIL_MAVEN_PUBLISH_CORE_BIN = $(UTIL_MAVEN_PUBLISH)/maven_publish_core.py
UTIL_COCOAPOD_FORMAT = $(UTILS_DIR)/cocoapod_format
UTIL_COCOAPOD_FORMAT_REQ = $(UTIL_COCOAPOD_FORMAT)/requirements.txt
UTIL_COCOAPOD_FORMAT_BRIDGE_BIN = $(UTIL_COCOAPOD_FORMAT)/cocoapod_format_bridge.py
UTIL_COCOAPOD_FORMAT_CORE_BIN = $(UTIL_COCOAPOD_FORMAT)/cocoapod_format_core.py
UTIL_COCOAPOD_PUBLISH = $(UTILS_DIR)/cocoapod_publish
UTIL_COCOAPOD_PUBLISH_REQ = $(UTIL_COCOAPOD_PUBLISH)/requirements.txt
UTIL_COCOAPOD_PUBLISH_BRIDGE_BIN = $(UTIL_COCOAPOD_PUBLISH)/cocoapod_publish_bridge.py
UTIL_COCOAPOD_PUBLISH_CORE_BIN = $(UTIL_COCOAPOD_PUBLISH)/cocoapod_publish_core.py
UTIL_BINTRAY_FORMAT = $(UTILS_DIR)/bintray_format
UTIL_BINTRAY_FORMAT_REQ = $(UTIL_BINTRAY_FORMAT)/requirements.txt
UTIL_BINTRAY_PUBLISH = $(UTILS_DIR)/bintray_publish
UTIL_BINTRAY_PUBLISH_REQ = $(UTIL_BINTRAY_PUBLISH)/requirements.txt
UTIL_BINTRAY_PUBLISH_ANDROID_BIN = $(UTIL_BINTRAY_PUBLISH)/bintray_publish_android.py
BUILD_DIR = $(MAKEFILE_DIR)/build
PIP ?= pip3

MANIFEST_GET_FUNC=$(or $(shell $(UTIL_MANIFEST_GET_BIN) $(1)),$(error "Can't get <$(1)> from Manifest.yml"))
# 版本号和包名设置
DEV_VERSION := 0.0.42-dev
# 实际使用的版本号：优先使用环境变量GOMOBILE_IPFS_VERSION，如不存在则使用DEV_VERSION
VERSION := $(or $(GOMOBILE_IPFS_VERSION),$(DEV_VERSION))
ANDROID_GROUP_ID := $(shell echo $(call MANIFEST_GET_FUNC,global.group_id) | tr . /)
ANDROID_CORE_ARTIFACT_ID := $(call MANIFEST_GET_FUNC,go_core.android.artifact_id)
ANDROID_APP_FILENAME := $(call MANIFEST_GET_FUNC,android_demo_app.filename)
ANDROID_MINIMUM_VERSION := $(call MANIFEST_GET_FUNC,android.min_sdk_version)
IOS_CORE_PACKAGE := $(call MANIFEST_GET_FUNC,go_core.ios.package)
IOS_APP_FILENAME := $(call MANIFEST_GET_FUNC,ios_demo_app.filename)


## Go相关目录和包设置 这些变量定义了Go代码的位置和将被编译成移动库的包路径。
# Go代码根目录
GO_DIR = $(MAKEFILE_DIR)/go
# 找出所有Go源文件
GO_SRC = $(shell find $(GO_DIR) -name \*.go)
# Go模块文件
GO_MOD_FILES = $(GO_DIR)/go.mod $(GO_DIR)/go.sum
# 核心包路径 - 这是要编译为移动库的Go包
CORE_PACKAGE = github.com/ipfs-shipyard/gomobile-ipfs/go/bind/core
EXT_PACKAGE ?=

GOMOBILE_OPT ?=
GOMOBILE_TARGET ?=
GOMOBILE_ANDROID_TARGET ?= android
GOMOBILE_IOS_TARGET ?= ios

# Android构建相关设置 这些定义了Android构建过程中的各种路径和文件位置。
ANDROID_DIR = $(MAKEFILE_DIR)/android
# Android源文件(排除.gitignore)
ANDROID_SRC = $(shell git ls-files $(ANDROID_DIR) | grep -v '.gitignore')
# Android构建目录
ANDROID_BUILD_DIR = $(BUILD_DIR)/android
# 中间构建目录
ANDROID_BUILD_DIR_INT = $(ANDROID_BUILD_DIR)/intermediates
ANDROID_BUILD_DIR_INT_CORE = $(ANDROID_BUILD_DIR_INT)/core
ANDROID_GOMOBILE_CACHE="$(ANDROID_BUILD_DIR_INT_CORE)/.gomobile-cache"
# 生成的AAR文件路径
ANDROID_CORE = $(ANDROID_BUILD_DIR_INT_CORE)/core.aar
ANDROID_BUILD_DIR_MAV = $(ANDROID_BUILD_DIR)/maven
ANDROID_BUILD_DIR_MAV_CORE = $(ANDROID_BUILD_DIR_MAV)/$(ANDROID_GROUP_ID)/$(ANDROID_CORE_ARTIFACT_ID)/$(VERSION)
ANDROID_OUTPUT_APK = $(ANDROID_DIR)/app/build/outputs/apk/release/app-release.apk
ANDROID_BUILD_DIR_APP = $(ANDROID_BUILD_DIR)/app/$(VERSION)
ANDROID_BUILD_DIR_APP_APK = $(ANDROID_BUILD_DIR_APP)/$(ANDROID_APP_FILENAME)-$(VERSION).apk

# iOS构建相关设置 这些定义了iOS构建过程中的各种路径和文件位置。
# iOS目录
IOS_DIR = $(MAKEFILE_DIR)/ios
# iOS源文件
IOS_SRC = $(shell git ls-files $(IOS_DIR) | grep -v '.gitignore')
IOS_BUILD_DIR = $(BUILD_DIR)/ios
IOS_BUILD_DIR_INT = $(IOS_BUILD_DIR)/intermediates
IOS_BUILD_DIR_INT_CORE = $(IOS_BUILD_DIR_INT)/core
IOS_GOMOBILE_CACHE="$(IOS_BUILD_DIR_INT_CORE)/.gomobile-cache"
# 生成的XCFramework路径
IOS_CORE = $(IOS_BUILD_DIR_INT_CORE)/Core.xcframework
IOS_BUILD_DIR_CCP = $(IOS_BUILD_DIR)/cocoapods
IOS_BUILD_DIR_CCP_CORE = $(IOS_BUILD_DIR_CCP)/$(IOS_CORE_PACKAGE)/$(VERSION)
IOS_WORKSPACE = $(IOS_DIR)/Example.xcworkspace
IOS_APP_PLIST = $(IOS_WORKSPACE)/release_export.plist
IOS_BUILD_DIR_APP = $(IOS_BUILD_DIR)/app/$(VERSION)
IOS_BUILD_DIR_APP_IPA = $(IOS_BUILD_DIR_APP)/$(IOS_APP_FILENAME)-$(VERSION).ipa
IOS_BUILD_DIR_INT_APP = $(IOS_BUILD_DIR_INT)/app
IOS_BUILD_DIR_INT_APP_IPA = $(IOS_BUILD_DIR_INT_APP)/ipa
IOS_BUILD_DIR_INT_APP_IPA_OUTPUT = $(IOS_BUILD_DIR_INT_APP_IPA)/Example.ipa
IOS_BUILD_DIR_INT_APP_ARCHIVE = $(IOS_BUILD_DIR_INT_APP)/archive
IOS_BUILD_DIR_INT_APP_ARCHIVE_OUTPUT = $(IOS_BUILD_DIR_INT_APP_ARCHIVE)/app-release.xcarchive

# 文档目录
DOC_DIR = $(MAKEFILE_DIR)/docs
ANDROID_DOC_DIR = $(DOC_DIR)/android
IOS_DOC_DIR = $(DOC_DIR)/ios

# 主要构建目标
.PHONY: help build_core build_core.android build_core.ios build_demo build_demo.android build_demo.ios clean clean.android clean.ios docgen docgen.android docgen.ios fail_on_dev publish publish_bridge publish_bridge.android publish_bridge.ios publish_core publish_core.android publish_core.ios publish_demo publish_demo.android publish_demo.ios re re.android re.ios test test_bridge test_bridge.android test_bridge.ios test_core

# 帮助信息
help:
	@echo 'Commands:'
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null \
		| awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' \
		| sort \
		| egrep -v -e '^[^[:alnum:]]' -e '^$@$$' \
		| grep -v / \
		| sed 's/^/	$(HELP_MSG_PREFIX)make /'

# 核心构建目标及依赖规则。这是主要的构建目标，调用它会同时构建Android和iOS的核心库。
build_core: build_core.android build_core.ios
# Android核心库构建
build_core.android: $(ANDROID_BUILD_DIR_MAV_CORE)
# Maven格式处理
$(ANDROID_BUILD_DIR_MAV_CORE): $(ANDROID_CORE) $(MANIFEST_FILE)
	@echo '------------------------------------'
	@echo '   Android Core: Maven formatting   '
	@echo '------------------------------------'
	# 检查并安装Python依赖
	if [ "$$($(PIP) freeze | grep -f $(UTIL_MAVEN_FORMAT_REQ) | wc -l)" != "$$(wc -l < $(UTIL_MAVEN_FORMAT_REQ))" ]; then \
		$(PIP) install -r $(UTIL_MAVEN_FORMAT_REQ); echo; \
	fi
	# 运行Maven格式化脚本
	$(UTIL_MAVEN_FORMAT_CORE_BIN) && touch $(ANDROID_BUILD_DIR_MAV_CORE)
	@echo 'Done!'

# GoMobile绑定生成AAR 这部分执行了从Go代码到Android AAR库的完整构建流程
$(ANDROID_CORE): $(ANDROID_BUILD_DIR_INT_CORE) $(GO_SRC) $(GO_MOD_FILES)
	@echo '------------------------------------'
	@echo '   Android Core: Gomobile binding   '
	@echo '------------------------------------'
	# 下载Go依赖
	cd $(GO_DIR) && go mod download
	# 初始化GoMobile
	cd $(GO_DIR) && go run golang.org/x/mobile/cmd/gomobile init
	# 创建缓存目录
	mkdir -p $(ANDROID_GOMOBILE_CACHE) android/libs
	# 运行GoMobile绑定命令，生成AAR
	GO111MODULE=on cd $(GO_DIR) && go run golang.org/x/mobile/cmd/gomobile bind \
		-o $(ANDROID_CORE) \
		-v $(GOMOBILE_OPT) \
		-cache $(ANDROID_GOMOBILE_CACHE) \
		-target=$(GOMOBILE_ANDROID_TARGET)$(GOMOBILE_TARGET) \
		-androidapi $(ANDROID_MINIMUM_VERSION) \
		$(CORE_PACKAGE) $(EXT_PACKAGE)
	touch $@
	cd $(GO_DIR) && go mod tidy
	@echo 'Done!'

$(ANDROID_BUILD_DIR_INT_CORE):
	mkdir -p $(ANDROID_BUILD_DIR_INT_CORE)
# iOS核心库构建
build_core.ios: $(IOS_BUILD_DIR_CCP_CORE)
# CocoaPod格式处理
$(IOS_BUILD_DIR_CCP_CORE): $(IOS_CORE) $(MANIFEST_FILE)
	@echo '------------------------------------'
	@echo '   iOS Core: CocoaPod formatting   '
	@echo '------------------------------------'
	# 检查并安装Python依赖
	if [ "$$($(PIP) freeze | grep -f $(UTIL_COCOAPOD_FORMAT_REQ) | wc -l)" != "$$(wc -l < $(UTIL_COCOAPOD_FORMAT_REQ))" ]; then \
		$(PIP) install -r $(UTIL_COCOAPOD_FORMAT_REQ); echo; \
	fi
	# 运行CocoaPod格式化脚本
	$(UTIL_COCOAPOD_FORMAT_CORE_BIN) && touch $(IOS_BUILD_DIR_CCP_CORE)
	@echo 'Done!'

# From https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile#hdr-Build_a_library_for_Android_and_iOS
# To generate a fat XCFramework that supports iOS, macOS, and macCatalyst for all supportec architectures (amd64 and arm64),
# specify -target ios,macos,maccatalyst
# we need to use `nowatchdog` tags, see https://github.com/libp2p/go-libp2p-connmgr/issues/98
# 来自 https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile#hdr-Build_a_library_for_Android_and_iOS
# 要生成支持iOS、macOS和macCatalyst的所有支持架构(amd64和arm64)的胖XCFramework，
# 需指定 -target ios,macos,maccatalyst
# 我们需要使用 `nowatchdog` 标签，参见 https://github.com/libp2p/go-libp2p-connmgr/issues/98
$(IOS_CORE): $(IOS_BUILD_DIR_INT_CORE) $(GO_SRC) $(GO_MOD_FILES)
	@echo '------------------------------------'
	@echo '     iOS Core: Gomobile binding     '
	@echo '------------------------------------'
	# 下载Go依赖
	cd $(GO_DIR) && go mod download
	# 安装gobind工具
	cd $(GO_DIR) && go install golang.org/x/mobile/cmd/gobind
	# 初始化GoMobile
	cd $(GO_DIR) && go run golang.org/x/mobile/cmd/gomobile init
	# 创建目录
	mkdir -p $(IOS_GOMOBILE_CACHE) ios/Frameworks
	# 运行GoMobile绑定命令，生成XCFramework
	cd $(GO_DIR) && go run golang.org/x/mobile/cmd/gomobile bind \
			-o $(IOS_CORE) \
			-tags 'nowatchdog' \
			$(GOMOBILE_OPT) \
			-cache $(IOS_GOMOBILE_CACHE) \
			-target=$(GOMOBILE_IOS_TARGET)$(GOMOBILE_TARGET) \
			$(CORE_PACKAGE) $(EXT_PACKAGE)
	touch $@
	cd $(GO_DIR) && go mod tidy
	@echo 'Done!'

$(IOS_BUILD_DIR_INT_CORE):
	@mkdir -p $(IOS_BUILD_DIR_INT_CORE)

build_demo: build_demo.android build_demo.ios

build_demo.android: $(ANDROID_BUILD_DIR_APP_APK)

$(ANDROID_BUILD_DIR_APP_APK): $(ANDROID_BUILD_DIR_APP) $(ANDROID_OUTPUT_APK) $(MANIFEST)
	@echo '------------------------------------'
	@echo '  Android Demo: apk path creation   '
	@echo '------------------------------------'
	cp $(ANDROID_OUTPUT_APK) $(ANDROID_BUILD_DIR_APP_APK)
	@echo 'Built .apk available in: $(ANDROID_BUILD_DIR_APP)'
	@echo 'Done!'

$(ANDROID_OUTPUT_APK): $(ANDROID_SRC) $(MANIFEST) $(ANDROID_BUILD_DIR_MAV_CORE)
	@echo '------------------------------------'
	@echo '   Android Demo: Gradle building    '
	@echo '------------------------------------'
	cd $(ANDROID_DIR) && ./gradlew app:build
	touch $(ANDROID_OUTPUT_APK)
	@echo 'Done!'

$(ANDROID_BUILD_DIR_APP):
	@mkdir -p $(ANDROID_BUILD_DIR_APP)

build_demo.ios: $(IOS_BUILD_DIR_APP_IPA)

$(IOS_BUILD_DIR_APP_IPA): $(IOS_BUILD_DIR_INT_APP_IPA_OUTPUT) $(MANIFEST)
	@echo '------------------------------------'
	@echo '    iOS Demo: Bintray formatting    '
	@echo '------------------------------------'
	if [ "$$($(PIP) freeze | grep -f $(UTIL_BINTRAY_FORMAT_REQ) | wc -l)" != "$$(wc -l < $(UTIL_BINTRAY_FORMAT_REQ))" ]; then \
		$(PIP) install -r $(UTIL_BINTRAY_FORMAT_REQ); echo; \
	fi
	#TODO
	@echo 'Done!'

$(IOS_BUILD_DIR_INT_APP_IPA_OUTPUT): $(IOS_BUILD_DIR_INT_APP_IPA) $(IOS_BUILD_DIR_INT_APP_ARCHIVE_OUTPUT)
	@echo '------------------------------------'
	@echo '   iOS Demo: XCode building ipa     '
	@echo '------------------------------------'
	xcodebuild -exportArchive \
		-archivePath $(IOS_BUILD_DIR_INT_APP_ARCHIVE_OUTPUT) \
		-exportOptionsPlist $(IOS_APP_PLIST) \
		-exportPath $(IOS_BUILD_DIR_INT_APP_IPA)
	touch $(IOS_BUILD_DIR_INT_APP_IPA_OUTPUT)
	@echo 'Done!'

$(IOS_BUILD_DIR_INT_APP_IPA):
	mkdir -p $(IOS_BUILD_DIR_INT_APP_IPA)

$(IOS_BUILD_DIR_INT_APP_ARCHIVE_OUTPUT): $(IOS_BUILD_DIR_INT_APP_ARCHIVE) $(IOS_CORE) $(IOS_SRC)
	@echo '------------------------------------'
	@echo '  iOS Demo: XCode building archive  '
	@echo '------------------------------------'
	xcodebuild archive \
		-workspace $(IOS_WORKSPACE) \
		-scheme Example \
		-configuration Release \
		-sdk iphoneos \
		-archivePath $(IOS_BUILD_DIR_INT_APP_ARCHIVE_OUTPUT)
	touch $(IOS_BUILD_DIR_INT_APP_ARCHIVE_OUTPUT)
	@echo 'Done!'

$(IOS_BUILD_DIR_INT_APP_ARCHIVE):
	@mkdir -p $(IOS_BUILD_DIR_INT_APP_ARCHIVE)

# 用于发布的目标
publish: publish_core publish_bridge publish_demo

publish.ios: publish_core.ios publish_bridge.ios

publish_core: publish_core.android publish_core.ios

publish_core.android: fail_on_dev build_core.android
	@echo '------------------------------------'
	@echo '   Android Core: Maven publishing   '
	@echo '------------------------------------'
	if [ "$$($(PIP) freeze | grep -f $(UTIL_MAVEN_PUBLISH_REQ) | wc -l)" != "$$(wc -l < $(UTIL_MAVEN_PUBLISH_REQ))" ]; then \
		$(PIP) install -r $(UTIL_MAVEN_PUBLISH_REQ); echo; \
	fi
	$(UTIL_MAVEN_PUBLISH_CORE_BIN)
	@echo 'Done!'

publish_core.ios: fail_on_dev build_core.ios
	@echo '------------------------------------'
	@echo '   iOS Core: CocoaPod publishing   '
	@echo '------------------------------------'
	if [ "$$($(PIP) freeze | grep -f $(UTIL_COCOAPOD_PUBLISH_REQ) | wc -l)" != "$$(wc -l < $(UTIL_COCOAPOD_PUBLISH_REQ))" ]; then \
		$(PIP) install -r $(UTIL_COCOAPOD_PUBLISH_REQ); echo; \
	fi
	$(UTIL_COCOAPOD_PUBLISH_CORE_BIN)
	@echo 'Done!'

publish_bridge: publish_bridge.android publish_bridge.ios

publish_bridge.android: fail_on_dev build_core.android
	@echo '------------------------------------'
	@echo '  Android Bridge: Maven publishing  '
	@echo '------------------------------------'
	@cd $(ANDROID_DIR) && ./gradlew bridge:bintrayUpload
	@echo 'Done!'

publish_bridge.ios: fail_on_dev build_core.ios
	@echo '------------------------------------'
	@echo '  iOS Bridge: CocoaPod publishing   '
	@echo '------------------------------------'
	if [ "$$($(PIP) freeze | grep -f $(UTIL_COCOAPOD_FORMAT_REQ) -f $(UTIL_COCOAPOD_PUBLISH_REQ) | wc -l)" != "$$(cat $(UTIL_COCOAPOD_FORMAT_REQ) $(UTIL_COCOAPOD_PUBLISH_REQ) | sort | uniq | wc -l )" ]; then \
		$(PIP) install -r $(UTIL_COCOAPOD_FORMAT_REQ) -r $(UTIL_COCOAPOD_PUBLISH_REQ); echo; \
	fi
	$(UTIL_COCOAPOD_FORMAT_BRIDGE_BIN) && $(UTIL_COCOAPOD_PUBLISH_BRIDGE_BIN)
	@echo 'Done!'

build_bridge.ios: fail_on_dev build_core.ios
	@echo '------------------------------------'
	@echo '  iOS Bridge: CocoaPod build        '
	@echo '------------------------------------'
	if [ "$$($(PIP) freeze | grep -f $(UTIL_COCOAPOD_FORMAT_REQ) -f $(UTIL_COCOAPOD_PUBLISH_REQ) | wc -l)" != "$$(cat $(UTIL_COCOAPOD_FORMAT_REQ) $(UTIL_COCOAPOD_PUBLISH_REQ) | sort | uniq | wc -l )" ]; then \
		$(PIP) install -r $(UTIL_COCOAPOD_FORMAT_REQ) -r $(UTIL_COCOAPOD_PUBLISH_REQ); echo; \
	fi
	$(UTIL_COCOAPOD_FORMAT_BRIDGE_BIN)
	@echo 'Done!'

publish_demo: publish_demo.android publish_demo.ios

publish_demo.android: fail_on_dev build_demo.android
	@echo '------------------------------------'
	@echo '  Android Demo: Bintray publishing  '
	@echo '------------------------------------'
	if [ "$$($(PIP) freeze | grep -f $(UTIL_BINTRAY_PUBLISH_REQ) | wc -l)" != "$$(wc -l < $(UTIL_BINTRAY_PUBLISH_REQ))" ]; then \
		$(PIP) install -r $(UTIL_BINTRAY_PUBLISH_REQ); echo; \
	fi
	$(UTIL_BINTRAY_PUBLISH_ANDROID_BIN)
	@echo 'Done!'

publish_demo.ios: fail_on_dev build_demo.ios
	@echo '------------------------------------'
	@echo '    iOS Demo: Bintray publishing    '
	@echo '------------------------------------'
	@if [ "$$($(PIP) freeze | grep -f $(UTIL_BINTRAY_PUBLISH_REQ) | wc -l)" != "$$(wc -l < $(UTIL_BINTRAY_PUBLISH_REQ))" ]; then \
		echo 'Installing pip dependencies:'; $(PIP) install -r $(UTIL_BINTRAY_PUBLISH_REQ); echo; \
	fi
	#TODO
	@echo 'Done!'

# 文档生成
docgen: docgen.android docgen.ios

docgen.android: $(ANDROID_DOC_DIR) build_core.android
	@echo '------------------------------------'
	@echo '   Android Bridge: Doc generation   '
	@echo '------------------------------------'
	cd $(ANDROID_DIR) && ./gradlew bridge:javadoc
	cp -rf $(ANDROID_DIR)/bridge/javadoc/* $(ANDROID_DOC_DIR)
	@echo 'Done!'

$(ANDROID_DOC_DIR):
	@mkdir -p $(ANDROID_DOC_DIR)

docgen.ios: $(IOS_DOC_DIR) build_core.ios
	@echo '------------------------------------'
	@echo '     iOS Bridge: Doc generation     '
	@echo '------------------------------------'
	cd $(IOS_DIR)/Bridge && \
		jazzy -o $(IOS_DOC_DIR) \
		--readme $(IOS_DIR)/../README.md \
		--module 'GomobileIPFS' \
		--title 'Gomobile-IPFS - iOS Bridge' \
		--github_url 'https://github.com/ipfs-shipyard/gomobile-ipfs' \
		--github-file-prefix 'https://github.com/ipfs-shipyard/gomobile-ipfs/tree/master/ios/Bridge'
	@echo 'Done!'

$(IOS_DOC_DIR):
	mkdir -p $(IOS_DOC_DIR)

# 运行所有测试
test: test_core test_bridge

test_bridge: test_bridge.android test_bridge.ios

test_bridge.android: build_core.android
	@echo '------------------------------------'
	@echo '   Android Bridge: running test     '
	@echo '------------------------------------'
	cd $(ANDROID_DIR) && ./gradlew bridge:test && \
	EMULATOR=$$(emulator -avd -list-avds | tail -n1); \
	if [ -z "$$EMULATOR" ]; then \
		>&2 echo "No emulator found to run the test";	\
		exit 1;	\
	fi;	\
	emulator -avd $$EMULATOR -no-boot-anim -no-window -no-snapshot-save -gpu swiftshader_indirect -noaudio & EMULATOR_PID=$$!; \
	adb wait-for-device shell 'while [[ -z $$(getprop sys.boot_completed) ]]; do sleep 1; done;'; \
	(cd $(ANDROID_DIR) && ./gradlew bridge:connectedAndroidTest) || \
	(kill $$EMULATOR_PID; exit 1) && \
	(kill $$EMULATOR_PID; echo 'Done!')

test_bridge.ios: build_core.ios
	@echo '------------------------------------'
	@echo '     iOS Bridge: running test       '
	@echo '------------------------------------'
	DESTINATION=$$(xcodebuild -showdestinations -project $(IOS_DIR)/Bridge/GomobileIPFS.xcodeproj -scheme GomobileIPFS | awk '/Ineligible destinations for/ {exit} {print}' | grep 'platform:iOS Simulator' | awk -F 'id:' '{print $$2}' | cut -d',' -f1 | tail -n1); \
	if [ -z "$$DESTINATION" ]; then \
		>&2 echo "No compatible simulator found to run the test";	\
		exit 1;	\
	fi;	\
	xcodebuild test -project $(IOS_DIR)/Bridge/GomobileIPFS.xcodeproj -scheme GomobileIPFS -sdk iphonesimulator -destination "platform=iOS Simulator,id=$$DESTINATION"
	@echo 'Done!'

# 运行Go核心测试
test_core:
	@echo '------------------------------------'
	@echo '       Go Core: running test        '
	@echo '------------------------------------'
	cd $(GO_DIR) && go test -v ./...
	@echo 'Done!'

# Misc rules
fail_on_dev:
	if [ "$(VERSION)" == "$(DEV_VERSION)" ]; then \
		>&2 echo "Can't publish a dev version: GOMOBILE_IPFS_VERSION env variable not set";	\
		exit 1; \
	fi

# 清理构建产物
clean: clean.android clean.ios
# 清理Android构建产物
clean.android:
	@echo '------------------------------------'
	@echo '  Android Core: removing build dir  '
	@echo '------------------------------------'
	rm -rf $(ANDROID_BUILD_DIR)

	# gomobile cache
ifneq (, $(wildcard $(ANDROID_GOMOBILE_CACHE)))
	chmod -R u+wx $(ANDROID_GOMOBILE_CACHE) && rm -rf $(ANDROID_GOMOBILE_CACHE)
endif
	@echo 'Done!'

clean.ios:
	@echo '------------------------------------'
	@echo '    iOS Core: removing build dir    '
	@echo '------------------------------------'
	rm -rf $(IOS_BUILD_DIR)

	# gomobile cache
ifneq (, $(wildcard $(IOS_GOMOBILE_CACHE)))
	chmod -R u+wx $(IOS_GOMOBILE_CACHE) && rm -rf $(IOS_GOMOBILE_CACHE)
endif
	@echo 'Done!'
