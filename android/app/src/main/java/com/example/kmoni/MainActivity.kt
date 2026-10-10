package com.example.kmoni

import android.os.Bundle
import android.os.Build
import android.provider.Settings
import android.widget.*
import androidx.appcompat.app.AppCompatActivity
import okhttp3.*
import org.json.JSONObject
import java.util.concurrent.TimeUnit
import java.util.concurrent.ScheduledExecutorService
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit.SECONDS

class MainActivity : AppCompatActivity() {
    private lateinit var server: EditText; private lateinit var token: EditText; private lateinit var status: TextView; private lateinit var details: TextView
    private val client=OkHttpClient.Builder().readTimeout(0,TimeUnit.MILLISECONDS).build(); private var ws: WebSocket?=null; private var retry=1000L; private val heartbeat: ScheduledExecutorService=Executors.newSingleThreadScheduledExecutor()
    private val prefs by lazy { getSharedPreferences("kmoni",0) }
    override fun onCreate(b: Bundle?) { super.onCreate(b); setContentView(R.layout.activity_main); server=findViewById(R.id.server); token=findViewById(R.id.token); status=findViewById(R.id.status); details=findViewById(R.id.details); server.setText(prefs.getString("server","")); token.setText(prefs.getString("token","")); findViewById<Button>(R.id.connect).setOnClickListener{connect()} }
    private fun connect() { val base=server.text.toString().trim().removeSuffix("/"); val secret=token.text.toString(); if(base.isBlank()||secret.isBlank()){status.text="Enter server URL and token";return}; prefs.edit().putString("server",base).putString("token",secret).apply(); ws?.cancel(); val wsUrl=base.replaceFirst("https://","wss://").replaceFirst("http://","ws://")+"/ws"; status.text="Connecting…"; ws=client.newWebSocket(Request.Builder().url(wsUrl).build(), object: WebSocketListener(){ override fun onOpen(w:WebSocket,r:Response){retry=1000; val o=JSONObject().apply{put("type","pair");put("token",secret);put("deviceId",deviceId());put("model",Build.MANUFACTURER+" "+Build.MODEL);put("android",Build.VERSION.RELEASE);put("app","0.1.0");put("rooted",hasRoot())};w.send(o.toString()); heartbeat.scheduleAtFixedRate({ val s=JSONObject().apply{put("type","status");put("rooted",hasRoot())}; w.send(s.toString()) },20,20,SECONDS)}; override fun onMessage(w:WebSocket,text:String){runOnUiThread{status.text="Connected"; details.text="Device: ${Build.MANUFACTURER} ${Build.MODEL}\nAndroid: ${Build.VERSION.RELEASE}\nRoot: ${hasRoot()}"}}; override fun onFailure(w:WebSocket,t:Throwable,r:Response?){runOnUiThread{status.text="Disconnected; retrying…"}; reconnect()}; override fun onClosed(w:WebSocket,c:Int,reason:String){reconnect()} }) }
    private fun reconnect(){ val delay=retry; retry=minOf(retry*2,60000); window.decorView.postDelayed({connect()},delay) }
    private fun deviceId()=Settings.Secure.getString(contentResolver,Settings.Secure.ANDROID_ID) ?: "unknown"
    private fun hasRoot()=arrayOf("/system/bin/su","/system/xbin/su").any{java.io.File(it).exists()}
    override fun onDestroy(){heartbeat.shutdownNow(); ws?.cancel(); super.onDestroy()}
}
