using System.IO;
using System.IO.Compression;
using System.Text;
using System.Windows;
using System.Windows.Threading;
using Microsoft.Web.WebView2.Core;
using Microsoft.Web.WebView2.Wpf;

namespace LasMdt;
internal static class NativeWebViewTest
{
 internal static void Start(){
  var view=new WebView2();var window=new Window{Title="LAS map response test",Width=500,Height=300,Content=view};
  Application.Current.MainWindow=window;
  void Fail(string message){File.WriteAllText("native-webview-error.txt",message);Application.Current.Shutdown(1);}
  var timeout=new DispatcherTimer{Interval=TimeSpan.FromSeconds(45)};timeout.Tick+=(_,_)=>Fail("WebView map response test timed out.");timeout.Start();
  window.Loaded+=async(_,_)=>{try{
   var environment=await CoreWebView2Environment.CreateAsync(null,Path.Combine(Path.GetTempPath(),"las-webview-test-"+Guid.NewGuid()));
   await view.EnsureCoreWebView2Async(environment);var core=view.CoreWebView2;
   core.AddWebResourceRequestedFilter("https://cad.lakesareasecurity.com/*",CoreWebView2WebResourceContext.All);
   using var compressed=new MemoryStream();using(var zip=new GZipStream(compressed,CompressionMode.Compress,true))zip.Write(new byte[]{26,3,10,1,65});
   var answer=new MapAnswer(200,"application/vnd.mapbox-vector-tile","gzip",Convert.ToBase64String(compressed.ToArray()));
   core.WebResourceRequested+=(_,e)=>{
    if(new Uri(e.Request.Uri).AbsolutePath=="/tile"){e.Response=TerminalWindow.MapResponse(environment,answer);return;}
    const string html="<html><body>Testing offline tile delivery<script>fetch('/tile').then(r=>r.arrayBuffer()).then(b=>chrome.webview.postMessage(Array.from(new Uint8Array(b)).join(',')==='26,3,10,1,65'?'pass':'wrong tile bytes')).catch(e=>chrome.webview.postMessage(String(e)))</script></body></html>";
    e.Response=environment.CreateWebResourceResponse(new MemoryStream(Encoding.UTF8.GetBytes(html)),200,"OK","Content-Type: text/html");
   };
   core.WebMessageReceived+=(_,e)=>{timeout.Stop();if(e.TryGetWebMessageAsString()=="pass"){view.Dispose();Application.Current.Shutdown(0);}else Fail(e.TryGetWebMessageAsString());};
   core.Navigate(AppPaths.Origin+"/map-response-test");
  }catch(Exception ex){Fail(ex.ToString());}};
  window.Show();
 }
}
