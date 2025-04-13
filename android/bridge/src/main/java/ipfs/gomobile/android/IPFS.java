/**
 * 核心用途：
 * 它提供了一个桥接层，连接Android应用与Go语言实现的IPFS节点，实际调用详见StartIPFS.java中
 * 它封装了IPFS节点的创建、配置、启动、停止和管理功能
 * 它提供了与IPFS节点交互的API，允许Android应用执行IPFS操作
 * 它处理了跨语言通信(Java-Go)的复杂性，使Android开发者能够简单地使用IPFS功能
 * 
 * 核心功能包括：
 * 初始化和管理IPFS仓库(存储IPFS数据的地方)
 * 创建和配置IPFS节点(包括网络、蓝牙、mDNS等设置)
 * 启动和停止IPFS节点服务
 * 提供配置管理(获取/设置IPFS配置)
 * 创建请求构建器，用于执行IPFS命令
 * 提供网关服务，允许通过HTTP访问IPFS内容
 */

package ipfs.gomobile.android; // 定义包名为ipfs.gomobile.android，这是Android平台上gomobile IPFS的实现

import android.content.Context;
import androidx.annotation.NonNull;

import java.io.File;
import java.lang.ref.WeakReference; // 导入弱引用，用于防止内存泄漏
import java.util.Objects;
import org.apache.commons.io.FilenameUtils;
import org.json.JSONObject;

// Import gomobile-ipfs core
// 导入Go语言编写的IPFS核心组件，这些是通过gomobile工具生成的Java绑定
import core.Core;
import core.Config;
import core.Repo;
import core.NodeConfig;
import core.Node;
import core.Shell;
import core.SockManager;
import ipfs.gomobile.android.bledriver.BleInterface;

/**
* IPFS is a class that wraps a go-ipfs node and its shell over UDS.
* IPFS是一个类，通过Unix域套接字(UDS)封装了go-ipfs节点及其Shell接口
*/
public class IPFS {

    private WeakReference<Context> context; // 使用弱引用存储Android上下文，防止内存泄漏
    // Paths
    private static final String defaultRepoPath = "/ipfs/repo"; // 默认仓库路径
    private final String absRepoPath; // 仓库的绝对路径
    private final String absSockPath; // 套接字的绝对路径

    // Go objects
    // Go语言对象，用于与Go实现的IPFS核心交互
    private static SockManager sockmanager; // 套接字管理器，静态共享
    private Node node; // IPFS节点实例
    private Repo repo; // IPFS仓库对象
    private Shell shell; // 与节点通信的Shell对象

    /**
    * Class constructor using defaultRepoPath "/ipfs/repo" on internal storage.
    * 类构造函数，使用内部存储上的默认仓库路径"/ipfs/repo"
    *
    * @param context The application context
    * @throws ConfigCreationException If the creation of the config failed
    * @throws RepoInitException If the initialization of the repo failed
    * @throws SockManagerException If the initialization of SockManager failed
    */
    public IPFS(@NonNull Context context)
        throws ConfigCreationException, RepoInitException, SockManagerException {
        this(context, defaultRepoPath, true); // 调用主构造函数，使用默认路径和内部存储
    }

    /**
    * Class constructor using repoPath passed as parameter on internal storage.
    * 类构造函数，使用传入的参数作为内部存储上的仓库路径
    *
    * @param context The application context
    * @param repoPath The path of the go-ipfs repo (relative to internal root)
    * @throws ConfigCreationException If the creation of the config failed
    * @throws RepoInitException If the initialization of the repo failed
    * @throws SockManagerException If the initialization of SockManager failed
    */
    public IPFS(@NonNull Context context, @NonNull String repoPath)
        throws ConfigCreationException, RepoInitException, SockManagerException {
        this(context, repoPath, true); // 调用主构造函数，使用指定路径和内部存储
    }

    /**
    * Class constructor using repoPath and storage location passed as parameters.
    * 类构造函数，使用传入的仓库路径和存储位置参数
    *
    * @param context The application context
    * @param repoPath The path of the go-ipfs repo (relative to internal/external root)
    * @param internalStorage true, if the desired storage location for the repo path is internal
    *                        false, if the desired storage location for the repo path is external
    * @throws ConfigCreationException If the creation of the config failed
    * @throws RepoInitException If the initialization of the repo failed
    * @throws SockManagerException If the initialization of SockManager failed
    */
    public IPFS(@NonNull Context context, @NonNull String repoPath, boolean internalStorage)
        throws ConfigCreationException, RepoInitException, SockManagerException {
        Objects.requireNonNull(context, "context should not be null"); // 检查参数非空
        Objects.requireNonNull(repoPath, "repoPath should not be null");

        String absPath;

        this.context = new WeakReference<>(context); // 使用弱引用存储上下文，防止内存泄漏

        if (internalStorage) {
            absPath = context.getFilesDir().getAbsolutePath(); // 获取内部存储的绝对路径
        } else {
            File externalDir = context.getExternalFilesDir(null); // 获取外部存储目录

            if (externalDir == null) {
                throw new RepoInitException("No external storage available"); // 外部存储不可用时抛出异常
            }
            absPath = externalDir.getAbsolutePath();
        }
        absRepoPath = FilenameUtils.normalizeNoEndSeparator(absPath + "/" + repoPath); // 构建并标准化仓库绝对路径

        synchronized (IPFS.class) { // 线程安全地初始化静态套接字管理器
            if (sockmanager == null) {
                try {
                    sockmanager = Core.newSockManager(context.getCacheDir().getAbsolutePath()); // 在缓存目录创建套接字管理器
                } catch (Exception e) {
                    throw new SockManagerException("Socket manager initialization failed", e);
                }
            }
        }

        try {
            absSockPath = sockmanager.newSockPath(); // 创建新的套接字路径，用于API通信
        } catch (Exception e) {
            throw new SockManagerException("API socket creation failed", e);
        }

        if (!Core.repoIsInitialized(absRepoPath)) { // 检查仓库是否已初始化
            Config config;

            try {
                config = Core.newDefaultConfig(); // 创建默认IPFS配置
            } catch (Exception e) {
                throw new ConfigCreationException("Config creation failed", e);
            }

            final File repoDir = new File(absRepoPath);
            if (!repoDir.exists()) {
                if (!repoDir.mkdirs()) { // 确保仓库目录存在
                    throw new RepoInitException("Repo directory creation failed: " + absRepoPath);
                }
            }
            try {
                Core.initRepo(absRepoPath, config); // 初始化IPFS仓库
            } catch (Exception e) {
                throw new RepoInitException("Repo initialization failed", e);
            }
        }
    }

    /**
    * Returns the repo absolute path as a string.
    * 以字符串形式返回仓库的绝对路径
    *
    * @return The repo absolute path
    */
    synchronized public String getRepoAbsolutePath() { // 线程安全方法
        return absRepoPath;
    }

    /**
    * Returns true if this IPFS instance is "started" by checking if the underlying go-ipfs node
    * is instantiated.
    * 通过检查底层go-ipfs节点是否已实例化，返回此IPFS实例是否"已启动"
    *
    * @return true, if this IPFS instance is started
    */
    synchronized public boolean isStarted() { // 线程安全方法
        return node != null; // 通过检查节点对象是否存在判断是否已启动
    }

    /**
    * Starts this IPFS instance. Also serve config Gateway and API located inside
    * the config (if any)
    * 启动此IPFS实例，同时提供配置中定义的网关和API服务(如果有)
    *
    * @throws NodeStartException If the node is already started or if its startup fails
    */
    synchronized public void start() throws NodeStartException { // 线程安全的启动方法
        if (isStarted()) {
            throw new NodeStartException("Node already started"); // 如果已启动则抛出异常
        }

        NodeConfig nodeConfig = Core.newNodeConfig(); // 创建节点配置对象
        nodeConfig.setBleDriver(new BleInterface(context.get(), true)); // 设置蓝牙驱动接口

        // set net driver
        // 在Android Q及以上版本中设置自定义网络驱动
        if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.Q) {
            NetDriver inet = new NetDriver();
            nodeConfig.setNetDriver(inet);
        }

        // set mdns locker driver
        // 设置多播DNS锁驱动，用于管理多播DNS服务
        MDNSLockerDriver imdnslocker = new MDNSLockerDriver(context.get());
        nodeConfig.setMDNSLocker(imdnslocker);

        try {
            openRepoIfClosed(); // 确保仓库已打开
            node = Core.newNode(repo, nodeConfig); // 创建IPFS节点(这是关键的Go调用)
            node.serveUnixSocketAPI(absSockPath); // 通过Unix套接字提供API服务

            // serve config Addresses API & Gateway
            // 启动配置中定义的API和网关服务
            node.serveConfig();
        } catch (Exception e) {
            throw new NodeStartException("Node start failed", e);
        }

        shell = Core.newUDSShell(absSockPath); // 创建Unix域套接字Shell对象，用于与节点通信
    }

    /**
    * Stops this IPFS instance.
    * 停止此IPFS实例
    *
    * @throws NodeStopException If the node is already stopped or if its stop fails
    */
    synchronized public void stop() throws NodeStopException { // 线程安全的停止方法
        if (!isStarted()) {
            throw new NodeStopException("Node not started yet"); // 如果未启动则抛出异常
        }

        try {
            node.close(); // 关闭IPFS节点
            node = null; // 清除节点引用
            repo = null; // 清除仓库引用
        } catch (Exception e) {
            throw new NodeStopException("Node stop failed", e);
        }
    }

    /**
    * Restarts this IPFS instance.
    * 重启此IPFS实例
    *
    * @throws NodeStopException If the node is already stopped or if its stop fails
    */
    synchronized public void restart() throws NodeStopException { // 线程安全的重启方法
        stop(); // 先停止节点
        try { start(); } catch(NodeStartException ignore) { /* Should never happen */ } // 然后启动节点，忽略可能的启动异常
    }

    /**
    * Gets the IPFS instance config as a JSON.
    * 以JSON形式获取IPFS实例配置
    *
    * @return The IPFS instance config as a JSON
    * @throws ConfigGettingException If the getting of the config failed
    * @see <a href="https://github.com/ipfs/go-ipfs/blob/master/docs/config.md">IPFS Config Doc</a>
    */
    synchronized public JSONObject getConfig() throws ConfigGettingException { // 线程安全方法
        try {
            openRepoIfClosed(); // 确保仓库已打开
            byte[] rawConfig = repo.getConfig().get(); // 获取原始配置数据
            return new JSONObject(new String(rawConfig)); // 将数据转换为JSON对象
        } catch (Exception e) {
            throw new ConfigGettingException("Config getting failed", e);
        }
    }

    /**
    * Sets JSON config passed as parameter as IPFS config or reset to default config (with a new
    * identity) if the config parameter is null.<br>
    * <b>A started instance must be restarted for its config to be applied.</b>
    * 将传入的JSON配置设置为IPFS配置，如果配置参数为null则重置为默认配置(具有新身份)
    * 已启动的实例必须重启才能应用其配置
    *
    * @param config The IPFS instance JSON config to set (if null, default config will be used)
    * @throws ConfigSettingException If the setting of the config failed
    * @see <a href="https://github.com/ipfs/go-ipfs/blob/master/docs/config.md">IPFS Config Doc</a>
    */
    synchronized public void setConfig(JSONObject config) throws ConfigSettingException { // 线程安全方法
        try {
            Config goConfig;

            if (config != null) {
                goConfig = Core.newConfig(config.toString().getBytes()); // 从JSON创建配置对象
            } else {
                goConfig = Core.newDefaultConfig(); // 创建默认配置
            }

            openRepoIfClosed(); // 确保仓库已打开
            repo.setConfig(goConfig); // 设置仓库配置
        } catch (Exception e) {
            throw new ConfigSettingException("Config setting failed", e);
        }
    }

    /**
    * Gets the JSON value associated to the key passed as parameter in the IPFS instance config.
    * 获取IPFS实例配置中与传入键关联的JSON值
    *
    * @param key The key associated to the value to get in the IPFS config
    * @return The JSON value associated to the key passed as parameter in the IPFS instance config
    * @throws ConfigGettingException If the getting of the config value failed
    * @see <a href="https://github.com/ipfs/go-ipfs/blob/master/docs/config.md">IPFS Config Doc</a>
    */
    synchronized public JSONObject getConfigKey(@NonNull String key) throws ConfigGettingException { // 线程安全方法
        Objects.requireNonNull(key, "key should not be null"); // 检查参数非空

        try {
            openRepoIfClosed(); // 确保仓库已打开
            byte[] rawValue = repo.getConfig().getKey(key); // 获取特定键的原始值
            return new JSONObject(new String(rawValue)); // 将数据转换为JSON对象
        } catch (Exception e) {
            throw new ConfigGettingException("Config value getting failed", e);
        }
    }

    /**
    * Sets JSON config value to the key passed as parameters in the IPFS instance config.<br>
    * <b>A started instance must be restarted for its config to be applied.</b>
    * 在IPFS实例配置中设置传入键的JSON配置值
    * 已启动的实例必须重启才能应用其配置
    *
    * @param key The key associated to the value to set in the IPFS instance config
    * @param value The JSON value associated to the key to set in the IPFS instance config
    * @throws ConfigSettingException If the setting of the config value failed
    * @see <a href="https://github.com/ipfs/go-ipfs/blob/master/docs/config.md">IPFS Config Doc</a>
    */
    synchronized public void setConfigKey(@NonNull String key, @NonNull JSONObject value)
        throws ConfigSettingException { // 线程安全方法
        Objects.requireNonNull(key, "key should not be null"); // 检查参数非空
        Objects.requireNonNull(value, "value should not be null");

        try {
            openRepoIfClosed(); // 确保仓库已打开
            Config ipfsConfig = repo.getConfig(); // 获取当前配置
            ipfsConfig.setKey(key, value.toString().getBytes()); // 设置特定键的值
            repo.setConfig(ipfsConfig); // 保存更新后的配置
        } catch (Exception e) {
            throw new ConfigSettingException("Config setting failed", e);
        }
    }

    /**
    * Creates and returns a RequestBuilder associated to this IPFS instance shell.
    * 创建并返回与此IPFS实例Shell关联的RequestBuilder
    *
    * @param command The command of the request
    * @return A RequestBuilder based on the command passed as parameter
    * @throws ShellRequestException If this IPFS instance is not started
    * @see <a href="https://docs.ipfs.io/reference/api/http/">IPFS API Doc</a>
    */
    synchronized public RequestBuilder newRequest(@NonNull String command)
        throws ShellRequestException { // 线程安全方法
        Objects.requireNonNull(command, "command should not be null"); // 检查参数非空

        if (!this.isStarted()) {
            throw new ShellRequestException("Shell request failed: node isn't started"); // 如果节点未启动则抛出异常
        }

        core.RequestBuilder requestBuilder = this.shell.newRequest(command); // 创建底层请求构建器
        return new RequestBuilder(requestBuilder); // 封装为Java端请求构建器
    }

    /**
    * Serves node gateway over the given multiaddr
    * 通过给定的多地址提供节点网关服务
    *
    * @param multiaddr The multiaddr to listen on
    * @param writable If true: will also support support `POST`, `PUT`, and `DELETE` methods.
    * @return The MultiAddr the node is serving on
    * @throws NodeListenException If the node failed to serve
    * @see <a href="https://docs.ipfs.io/concepts/ipfs-gateway/#gateway-providers">IPFS Doc</a>
    */
    synchronized public String serveGatewayMultiaddr(@NonNull String multiaddr, @NonNull Boolean writable) throws NodeListenException { // 线程安全方法
        try {
            return node.serveGatewayMultiaddr(multiaddr, writable); // 在指定多地址上提供网关服务
        } catch (Exception e) {
            throw new NodeListenException("failed to listen on gateway", e);
        }
    }

    /**
    * Internal helper that opens the repo if it is closed.
    * 内部辅助方法，如果仓库已关闭则打开它
    *
    * @throws RepoOpenException If the opening of the repo failed
    */
    synchronized private void openRepoIfClosed() throws RepoOpenException { // 线程安全的内部辅助方法
        if (repo == null) { // 检查仓库是否为空
            try {
                repo = Core.openRepo(absRepoPath); // 打开仓库
            } catch (Exception e) {
                throw new RepoOpenException("Repo opening failed", e);
            }
        }
    }

    // Exceptions
    // 自定义异常类，用于精确报告各种操作过程中可能发生的错误
    public static class ExtraOptionException extends Exception {
        ExtraOptionException(String message, Throwable err) { super(message, err); }
    }

    public static class NodeListenException extends Exception {
        NodeListenException(String message, Throwable err) { super(message, err); }
    }

    public static class ConfigCreationException extends Exception {
        ConfigCreationException(String message, Throwable err) { super(message, err); }
    }

    public static class ConfigGettingException extends Exception {
        ConfigGettingException(String message, Throwable err) { super(message, err); }
    }

    public static class ConfigSettingException extends Exception {
        ConfigSettingException(String message, Throwable err) { super(message, err); }
    }

    public static class NodeStartException extends Exception {
        NodeStartException(String message) { super(message); }
        NodeStartException(String message, Throwable err) { super(message, err); }
    }

    public static class NodeStopException extends Exception {
        NodeStopException(String message) { super(message); }
        NodeStopException(String message, Throwable err) { super(message, err); }
    }

    public static class SockManagerException extends Exception {
        SockManagerException(String message, Throwable err) { super(message, err); }
    }

    public static class RepoInitException extends Exception {
        RepoInitException(String message) { super(message); }
        RepoInitException(String message, Throwable err) { super(message, err); }
    }

    public static class RepoOpenException extends Exception {
        RepoOpenException(String message, Throwable err) { super(message, err); }
    }

    public static class ShellRequestException extends Exception {
        ShellRequestException(String message) { super(message); }
    }
}