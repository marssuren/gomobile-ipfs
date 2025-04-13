/**
 * 这个类是专门用于异步启动IPFS实例的任务类。
 * 它作为一个桥梁，连接了Android应用的UI层(MainActivity)和底层的IPFS功能库。
 * 具体来说，它的主要功能有：
 * 封装异步操作：利用AsyncTask在后台线程启动IPFS节点，避免阻塞UI线程，保持应用响应性
 * 安全处理生命周期：使用WeakReference引用Activity，防止内存泄漏，并在Activity不可用时取消操作
 * 完整的启动流程：
 * 创建IPFS实例
 * 启动IPFS节点
 * 验证节点工作正常(通过id命令)
 * 将实例返回给Activity供后续使用
 */

package ipfs.gomobile.example; // 定义包名，表明这是IPFS示例应用的一部分

import android.os.AsyncTask; // 导入AsyncTask用于在后台线程执行耗时操作
import android.util.Log; // 导入日志功能

import org.json.JSONObject; // 用于处理JSON数据

import java.lang.ref.WeakReference; // 导入弱引用，防止内存泄漏
import java.util.ArrayList; // 用于存储集合数据

import ipfs.gomobile.android.IPFS; // 导入IPFS主类，这是我们的核心功能库

// 定义AsyncTask子类，用于在后台线程启动IPFS节点
// <Void, Void, String>表示：无输入参数、无进度更新、返回字符串结果(节点ID或错误信息)
final class StartIPFS extends AsyncTask<Void, Void, String> {
    private static final String TAG = "StartIPFS"; // 定义日志标签，用于标识来自此类的日志

    private final WeakReference<MainActivity> activityRef; // 使用弱引用持有Activity，防止内存泄漏
    private boolean backgroundError; // 标记后台操作是否发生错误

    // 构造函数，接收MainActivity并存储为弱引用
    StartIPFS(MainActivity activity) {
        activityRef = new WeakReference<>(activity);
    }

    @Override
    protected void onPreExecute() {
    } // 任务执行前的操作，此处为空实现

    @Override
    protected String doInBackground(Void... v) { // 在后台线程执行的耗时操作
        MainActivity activity = activityRef.get(); // 从弱引用获取Activity实例
        if (activity == null || activity.isFinishing()) { // 检查Activity是否仍然有效
            cancel(true); // 如果Activity已销毁或正在结束，取消任务
            return null;
        }

        try {
            IPFS ipfs = new IPFS(activity.getApplicationContext()); // 创建IPFS实例，使用应用上下文而非Activity上下文
            ipfs.start(); // 启动IPFS节点，这是耗时操作

            ArrayList<JSONObject> jsonList = ipfs.newRequest("id").sendToJSONList(); // 发送"id"命令获取节点信息

            activity.setIpfs(ipfs); // 将IPFS实例保存到Activity中供后续使用
            return jsonList.get(0).getString("ID"); // 返回节点的唯一标识符(PeerID)作为成功结果
        } catch (Exception err) {
            backgroundError = true; // 设置错误标志
            return MainActivity.exceptionToString(err); // 返回异常的字符串表示作为错误结果
        }
    }

    protected void onPostExecute(String result) { // 后台任务完成后在UI线程执行的方法
        MainActivity activity = activityRef.get(); // 再次从弱引用获取Activity
        if (activity == null || activity.isFinishing())
            return; // 检查Activity是否仍有效，无效则不进行UI更新

        if (backgroundError) { // 根据错误标志决定显示错误还是成功结果
            activity.displayPeerIDError(result); // 显示错误信息
            Log.e(TAG, "IPFS start error: " + result); // 记录错误日志
        } else {
            activity.displayPeerIDResult(result); // 显示节点ID
            Log.i(TAG, "Your PeerID is: " + result); // 记录信息日志
        }
    }
}
