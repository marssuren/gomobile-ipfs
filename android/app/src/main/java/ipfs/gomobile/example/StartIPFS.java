package ipfs.gomobile.example;

import android.os.AsyncTask;
import android.util.Log;

import org.json.JSONObject;

import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.PrintWriter;
import java.io.StringWriter;
import java.lang.ref.WeakReference;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.Locale;

import ipfs.gomobile.android.IPFS;

final class StartIPFS extends AsyncTask<Void, Void, String> {
    private static final String TAG = "StartIPFS";

    private final WeakReference<MainActivity> activityRef;
    private boolean backgroundError;
    private long startTime;

    StartIPFS(MainActivity activity) {
        activityRef = new WeakReference<>(activity);
    }

    @Override
    protected void onPreExecute() {
        startTime = System.currentTimeMillis();
        Log.d(TAG, "开始IPFS节点启动流程");
    }

    @Override
    protected String doInBackground(Void... v) {
        MainActivity activity = activityRef.get();
        if (activity == null || activity.isFinishing()) {
            cancel(true);
            return null;
        }

        // 创建应用级日志文件
        File logDir = new File(activity.getFilesDir(), "ipfs_logs");
        if (!logDir.exists()) {
            logDir.mkdirs();
        }
        
        SimpleDateFormat sdf = new SimpleDateFormat("yyyyMMdd_HHmmss", Locale.getDefault());
        File logFile = new File(logDir, "ipfs_app_" + sdf.format(new Date()) + ".log");
        
        try (FileOutputStream fos = new FileOutputStream(logFile)) {
            writeLog(fos, "====== 开始IPFS节点启动 ======");
            writeLog(fos, "时间: " + new Date());
            writeLog(fos, "设备型号: " + android.os.Build.MODEL);
            writeLog(fos, "Android版本: " + android.os.Build.VERSION.RELEASE);
            writeLog(fos, "应用版本: " + activity.getPackageName() + " / " + BuildConfig.VERSION_NAME);
            
            try {
                writeLog(fos, "正在创建IPFS实例...");
                IPFS ipfs = new IPFS(activity.getApplicationContext());
                writeLog(fos, "IPFS实例创建成功，路径: " + ipfs.getRepoAbsolutePath());
                
                writeLog(fos, "正在启动IPFS节点...");
                long beforeStart = System.currentTimeMillis();
                try {
                    ipfs.start();
                    long afterStart = System.currentTimeMillis();
                    writeLog(fos, "IPFS节点启动成功! 耗时: " + (afterStart - beforeStart) + "ms");
                } catch (Exception e) {
                    writeLog(fos, "IPFS节点启动失败!");
                    logException(fos, e);
                    throw e;
                }
                
                writeLog(fos, "正在获取节点ID...");
                ArrayList<JSONObject> jsonList = ipfs.newRequest("id").sendToJSONList();
                String peerId = jsonList.get(0).getString("ID");
                writeLog(fos, "成功获取节点ID: " + peerId);
                
                // 记录libp2p地址
                try {
                    ArrayList<String> addresses = new ArrayList<>();
                    if (jsonList.get(0).has("Addresses")) {
                        for (int i = 0; i < jsonList.get(0).getJSONArray("Addresses").length(); i++) {
                            addresses.add(jsonList.get(0).getJSONArray("Addresses").getString(i));
                        }
                    }
                    writeLog(fos, "节点地址: ");
                    for (String addr : addresses) {
                        writeLog(fos, "  - " + addr);
                    }
                } catch (Exception e) {
                    writeLog(fos, "无法获取节点地址: " + e.getMessage());
                }

                activity.setIpfs(ipfs);
                return peerId;
            } catch (Exception err) {
                backgroundError = true;
                writeLog(fos, "====== IPFS启动中发生严重错误 ======");
                logException(fos, err);
                
                // 尝试获取额外的诊断信息
                writeLog(fos, "\n====== 诊断信息 ======");
                try {
                    // 检查存储权限
                    File testWrite = new File(activity.getFilesDir(), "test_write.tmp");
                    boolean canWrite = testWrite.createNewFile();
                    writeLog(fos, "存储写入权限: " + canWrite);
                    if (testWrite.exists()) {
                        testWrite.delete();
                    }
                    
                    // 检查网络状态
                    boolean networkAvailable = activity.isNetworkAvailable();
                    writeLog(fos, "网络连接状态: " + (networkAvailable ? "可用" : "不可用"));
                    
                    // 获取可用内存
                    Runtime rt = Runtime.getRuntime();
                    long maxMemory = rt.maxMemory() / (1024 * 1024);
                    long freeMemory = rt.freeMemory() / (1024 * 1024);
                    long totalMemory = rt.totalMemory() / (1024 * 1024);
                    writeLog(fos, String.format("内存状态: 最大=%dMB, 已用=%dMB, 可用=%dMB", 
                            maxMemory, totalMemory - freeMemory, freeMemory));
                } catch (Exception e) {
                    writeLog(fos, "获取诊断信息时出错: " + e.getMessage());
                }
                
                return MainActivity.exceptionToString(err);
            }
        } catch (IOException e) {
            Log.e(TAG, "写入日志文件失败: " + e.getMessage());
            backgroundError = true;
            return "日志写入失败: " + e.getMessage() + "\n原始错误: " + e.toString();
        }
    }

    protected void onPostExecute(String result) {
        MainActivity activity = activityRef.get();
        if (activity == null || activity.isFinishing()) return;

        long totalTime = System.currentTimeMillis() - startTime;

        if (backgroundError) {
            activity.displayPeerIDError(result);
            Log.e(TAG, "IPFS启动失败，耗时 " + totalTime + "ms，错误: " + result);
        } else {
            activity.displayPeerIDResult(result);
            Log.i(TAG, "IPFS启动成功，耗时 " + totalTime + "ms，节点ID: " + result);
        }
    }
    
    // 写入日志工具方法
    private void writeLog(FileOutputStream fos, String message) throws IOException {
        String timestamp = new SimpleDateFormat("HH:mm:ss.SSS", Locale.getDefault()).format(new Date());
        String logLine = timestamp + " - " + message + "\n";
        fos.write(logLine.getBytes());
        Log.d(TAG, message);
    }
    
    // 记录异常详情
    private void logException(FileOutputStream fos, Exception e) throws IOException {
        writeLog(fos, "错误类型: " + e.getClass().getName());
        writeLog(fos, "错误消息: " + e.getMessage());
        
        StringWriter sw = new StringWriter();
        PrintWriter pw = new PrintWriter(sw);
        e.printStackTrace(pw);
        String stackTrace = sw.toString();
        
        writeLog(fos, "堆栈跟踪:\n" + stackTrace);
        
        // 记录原始异常
        Throwable cause = e.getCause();
        if (cause != null) {
            writeLog(fos, "原始错误: " + cause.getMessage());
            sw = new StringWriter();
            pw = new PrintWriter(sw);
            cause.printStackTrace(pw);
            writeLog(fos, "原始堆栈:\n" + sw.toString());
        }
    }
}
