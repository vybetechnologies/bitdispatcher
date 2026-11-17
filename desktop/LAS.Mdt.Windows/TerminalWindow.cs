using System.Diagnostics;
using System.IO;
using System.IO.Compression;
using System.Net;
using System.Net.Http;
using System.Reflection;
using System.Runtime.InteropServices;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Interop;
using System.Windows.Media;
using System.Windows.Threading;
using Microsoft.Web.WebView2.Core;
using Microsoft.Web.WebView2.Wpf;

namespace LasMdt;
internal sealed class TerminalWindow : Window
{
 private readonly WebView2 terminal=new();
 private readonly ProtectedStore storage=new(Path.Combine(AppPaths.Data,"offline"));
 private readonly MapHost maps=new();
 private readonly TextBlock status=new(){Text="Starting LAS MDT…",Margin=new Thickness(10,4,10,4),Foreground=Brushes.LightGray};
 private readonly Button updateButton=new(){Content="Install update",Visibility=Visibility.Collapsed,Margin=new Thickness(8,0,8,0)};
 private readonly Dictionary<string,byte[]> assets=new(StringComparer.Ordinal);
 private Release? release;
 private bool updating;
 private bool connectionDown;
 [DllImport("dwmapi.dll")]private static extern int DwmSetWindowAttribute(IntPtr handle,int attribute,ref int value,int size);
 internal TerminalWindow()
 {
  Title="Lakes Area Security MDT";Width=1440;Height=900;MinWidth=800;MinHeight=540;Background=new SolidColorBrush(Color.FromRgb(2,6,23));
  var dock=new DockPanel();Content=dock;
  var menu=new Menu{Background=SystemColors.MenuBrush,Foreground=SystemColors.MenuTextBrush};
  var terminalMenu=new MenuItem{Header="_Terminal"};
  Add(terminalMenu,"_MDT",()=>terminal.CoreWebView2?.Navigate(AppPaths.Origin+"/#/mdt"));
  Add(terminalMenu,"_Sync now",()=>terminal.CoreWebView2?.ExecuteScriptAsync("window.dispatchEvent(new Event('las:sync'))"));
  Add(terminalMenu,"Offline _maps",async()=>{try{await DownloadMap();}catch(Exception ex){MessageBox.Show(ex.Message,"Offline maps");}});
  terminalMenu.Items.Add(new Separator());Add(terminalMenu,"E_xit",Close);
  var viewMenu=new MenuItem{Header="_View"};Add(viewMenu,"_Full screen (F11)",ToggleFullScreen);Add(viewMenu,"_Reload",()=>terminal.Reload());
  var helpMenu=new MenuItem{Header="_Help"};Add(helpMenu,"Check for _updates",async()=>await CheckUpdates(true));
  Add(helpMenu,"_About",()=>MessageBox.Show("Lakes Area Security MDT\nVersion "+Updates.Current+"\nIndependent Windows application\nSaved offline work syncs when connected.\nRecorded maps © OpenStreetMap contributors.","About LAS MDT",MessageBoxButton.OK,MessageBoxImage.Information));
  menu.Items.Add(terminalMenu);menu.Items.Add(viewMenu);menu.Items.Add(helpMenu);DockPanel.SetDock(menu,Dock.Top);dock.Children.Add(menu);
  var footer=new DockPanel{Background=new SolidColorBrush(Color.FromRgb(15,23,42))};DockPanel.SetDock(footer,Dock.Bottom);DockPanel.SetDock(updateButton,Dock.Right);footer.Children.Add(updateButton);footer.Children.Add(status);dock.Children.Add(footer);dock.Children.Add(terminal);
  updateButton.Click+=async(_,_)=>await InstallUpdate();
  Loaded+=async(_,_)=>await Start();
  SourceInitialized+=(_,_)=>{var value=1;DwmSetWindowAttribute(new WindowInteropHelper(this).Handle,20,ref value,sizeof(int));};
  PreviewKeyDown+=(_,e)=>{if(e.Key==Key.F11){ToggleFullScreen();e.Handled=true;}};
  Closed+=(_,_)=>{maps.Dispose();terminal.Dispose();};
 }
 private static void Add(MenuItem parent,string title,Action action){var item=new MenuItem{Header=title};item.Click+=(_,_)=>action();parent.Items.Add(item);}
 private void ToggleFullScreen(){if(WindowStyle==WindowStyle.None){WindowStyle=WindowStyle.SingleBorderWindow;WindowState=WindowState.Normal;}else{WindowStyle=WindowStyle.None;WindowState=WindowState.Maximized;}}
 private async Task Start()
 {
  try {
   using(var stream=Assembly.GetExecutingAssembly().GetManifestResourceStream("WebAssets.zip")??throw new IOException("The bundled MDT interface is missing."))
   using(var archive=new ZipArchive(stream)){foreach(var entry in archive.Entries.Where(e=>!e.FullName.EndsWith('/'))){using var file=entry.Open();using var data=new MemoryStream();file.CopyTo(data);assets["/"+entry.FullName.Replace('\\','/')]=data.ToArray();}}
   var environment=await CoreWebView2Environment.CreateAsync(null,Path.Combine(AppPaths.Data,"browser"));
   await terminal.EnsureCoreWebView2Async(environment);
   var core=terminal.CoreWebView2;core.Settings.AreDevToolsEnabled=false;core.Settings.AreDefaultContextMenusEnabled=false;core.Settings.IsStatusBarEnabled=false;
   core.AddWebResourceRequestedFilter(AppPaths.Origin+"/*",CoreWebView2WebResourceContext.All);
   core.WebResourceRequested+=ResourceRequested;
   core.NavigationStarting+=(_,e)=>{if(!TerminalPage(e.Uri)){e.Cancel=true;if(!TrustedOrigin(e.Uri))OpenExternal(e.Uri);}};
   core.NewWindowRequested+=(_,e)=>{e.Handled=true;if(TerminalPage(e.Uri))core.Navigate(e.Uri);else if(!TrustedOrigin(e.Uri))OpenExternal(e.Uri);};
   core.PermissionRequested+=(_,e)=>{if(TrustedOrigin(e.Uri)&&e.PermissionKind==CoreWebView2PermissionKind.Geolocation)e.State=CoreWebView2PermissionState.Allow;};
   core.WebMessageReceived+=Bridge;
   await core.AddScriptToExecuteOnDocumentCreatedAsync("""
    if(window.top===window && location.origin==='https://cad.lakesareasecurity.com' && (location.pathname==='/'||location.pathname==='/index.html')){
     const waiting=new Map();let seq=0;
     window.chrome.webview.addEventListener('message',e=>{const r=e.data;if(!r||!waiting.has(r.id))return;const p=waiting.get(r.id);waiting.delete(r.id);clearTimeout(p.timer);r.error?p.reject(new Error(r.error)):p.resolve(r.value);});
     window.lasDesktop={version:'1.0.5',call:(op,key,value)=>new Promise((resolve,reject)=>{const id=++seq;const timer=setTimeout(()=>{waiting.delete(id);reject(new Error('Windows storage did not respond. Keep this work open and retry.'));},op==='downloadMap'?900000:120000);waiting.set(id,{resolve,reject,timer});window.chrome.webview.postMessage({id,op,key,value});})};
    }
    """);
   core.Navigate(AppPaths.Origin+"/#/mdt");
   status.Text="LAS MDT "+Updates.Current+" · Windows · sign in online once to prepare offline work";
   var timer=new DispatcherTimer{Interval=TimeSpan.FromHours(1)};timer.Tick+=async(_,_)=>await CheckUpdates(false);timer.Start();
   await CheckUpdates(false);
  }catch(WebView2RuntimeNotFoundException){MessageBox.Show("The Microsoft Edge WebView2 Runtime is required. Install the Evergreen Runtime from Microsoft's official WebView2 download page, then reopen LAS MDT.","LAS MDT");OpenExternal("https://developer.microsoft.com/microsoft-edge/webview2/");}
  catch(Exception ex){status.Text="Could not start the terminal: "+ex.Message;}
 }
 private static bool TrustedOrigin(string value)=>Uri.TryCreate(value,UriKind.Absolute,out var u)&&u.Scheme=="https"&&u.Host=="cad.lakesareasecurity.com"&&u.Port==443;
 internal static bool TerminalPage(string value){
  if(!TrustedOrigin(value))return false;
  var u=new Uri(value);
  if(IsAttachment(u.AbsolutePath))return true;
  if(u.AbsolutePath!="/"&&u.AbsolutePath!="/index.html")return false;
  var route=u.Fragment;
  return route is "" or "#" or "#/" or "#/mdt" or "#/mdt/reports" || System.Text.RegularExpressions.Regex.IsMatch(route,"^#/mdt/reports/[A-Za-z0-9_-]+$");
 }
 private static void OpenExternal(string value){if(Uri.TryCreate(value,UriKind.Absolute,out var u)&&u.Scheme=="https")Process.Start(new ProcessStartInfo(value){UseShellExecute=true});}
 private async void ResourceRequested(object? sender,CoreWebView2WebResourceRequestedEventArgs e)
 {
  var uri=new Uri(e.Request.Uri);var path=Uri.UnescapeDataString(uri.AbsolutePath);
  var asset=path=="/"||path=="/index.html"?"/index.html":path;
  if(assets.TryGetValue(asset,out var bytes)){
   var type=Path.GetExtension(asset).ToLowerInvariant() switch {".html"=>"text/html; charset=utf-8",".js" or ".mjs"=>"text/javascript",".css"=>"text/css",".svg"=>"image/svg+xml",".webp"=>"image/webp",".woff2"=>"font/woff2",_=>"application/octet-stream"};
   var csp=asset=="/index.html"?"\r\nContent-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' blob:; worker-src 'self' blob:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'":"";
   e.Response=terminal.CoreWebView2.Environment.CreateWebResourceResponse(new MemoryStream(bytes,false),200,"OK","Content-Type: "+type+"\r\nCache-Control: no-store"+csp);return;
  }
  if(IsAttachment(path)){
   var deferral=e.GetDeferral();
   try{
    var file=await Attachment(uri);
    if(file!=null)e.Response=terminal.CoreWebView2.Environment.CreateWebResourceResponse(new MemoryStream(Convert.FromBase64String(file.Data)),200,"Saved attachment","Content-Type: "+file.Type+"\r\nX-Content-Type-Options: nosniff\r\nContent-Security-Policy: sandbox; default-src 'none'; img-src 'self'; media-src 'self'\r\nCache-Control: no-store");
   }catch{ /* Ordinary network handling reports an uncached or unauthorized attachment. */ }
   finally{deferral.Complete();}
   return;
  }
  if(path.StartsWith("/api/map/tiles/",StringComparison.Ordinal)||path.StartsWith("/api/map/fonts/",StringComparison.Ordinal)){
   using var deferral=e.GetDeferral();
   try{
    var result=await maps.Request(uri.PathAndQuery,null);
    if(result.Status==404)return;
    // WebView2 synthetic response streams must contain decoded tile bytes.
    e.Response=MapResponse(terminal.CoreWebView2.Environment,result);
   }catch{ /* The normal network map remains available if the local package is not prepared. */ }
  }
 }
 internal static CoreWebView2WebResourceResponse MapResponse(CoreWebView2Environment environment,MapAnswer result){
  var decoded=MapHost.ResponseBytes(result);
  return environment.CreateWebResourceResponse(new MemoryStream(decoded,false),result.Status,"Offline recorded map","Content-Type: "+result.ContentType+"\r\nCache-Control: no-store\r\n");
 }
 private async void Bridge(object? sender,CoreWebView2WebMessageReceivedEventArgs e)
 {
  if(!TrustedOrigin(e.Source)||!(new Uri(e.Source).AbsolutePath is "/" or "/index.html"))return;
  long id=0;
  try{
   using var json=JsonDocument.Parse(e.WebMessageAsJson);var message=json.RootElement;
   id=message.GetProperty("id").GetInt64();var op=message.GetProperty("op").GetString();var key=message.TryGetProperty("key",out var k)?k.GetString()??"":"";
   object? value=null;
   switch(op){
    case "read":value=await storage.Read(key);break;
    case "write":await storage.Write(key,message.GetProperty("value").GetString()??"");break;
    case "delete":await storage.Delete(key);break;
    case "status":status.Text=message.GetProperty("value").GetString()??"LAS MDT";connectionDown=status.Text.Contains("OFFLINE —",StringComparison.Ordinal);break;
    case "cacheFile":if(!IsAttachment(new Uri(AppPaths.Origin+key).AbsolutePath))throw new InvalidOperationException("Not a supported attachment.");value=await Attachment(new Uri(AppPaths.Origin+key),true)!=null;break;
    case "map":value=await maps.Request(key,message.TryGetProperty("value",out var body)?body.GetRawText():null);break;
    case "downloadMap":await DownloadMap();value=File.Exists(MapHost.MapPath);break;
    case "clearCookies":terminal.CoreWebView2.CookieManager.DeleteAllCookies();break;
    default:throw new InvalidOperationException("Unsupported Windows MDT operation.");
   }
   terminal.CoreWebView2.PostWebMessageAsJson(JsonSerializer.Serialize(new{id,value}));
  }catch(Exception ex){terminal.CoreWebView2.PostWebMessageAsJson(JsonSerializer.Serialize(new{id,error=ex.Message}));}
 }
 private sealed record SavedAttachment(string Type,string Data);
 private static bool IsAttachment(string path)=>System.Text.RegularExpressions.Regex.IsMatch(path,"^/api/(incidents/[A-Za-z0-9_-]+/files|comms/files)/[A-Za-z0-9_-]+$");
 private async Task<SavedAttachment?> Attachment(Uri uri,bool warm=false)
 {
  var identity=await storage.Read("identity");if(identity==null)return null;
  using var json=JsonDocument.Parse(identity);var session=json.RootElement.GetProperty("session");
  var owner=session.GetProperty("username").GetString();var unit=session.GetProperty("mdtUnit").GetString();
  var key="attachment:"+owner+"|"+unit+":"+uri.AbsolutePath;
  if(warm&&storage.Exists(key))return new SavedAttachment("","");
  if(!connectionDown){
   try{
    using var handler=new HttpClientHandler{AllowAutoRedirect=false,CookieContainer=new CookieContainer()};
    foreach(var cookie in await terminal.CoreWebView2.CookieManager.GetCookiesAsync(AppPaths.Origin))handler.CookieContainer.Add(new Uri(AppPaths.Origin),new Cookie(cookie.Name,cookie.Value,cookie.Path));
    using var http=new HttpClient(handler){Timeout=TimeSpan.FromSeconds(12)};
    using var response=await http.GetAsync(uri,HttpCompletionOption.ResponseHeadersRead);
    if(response.StatusCode==HttpStatusCode.Forbidden||response.StatusCode==HttpStatusCode.Unauthorized){await storage.Delete(key);return null;}
    response.EnsureSuccessStatusCode();
    if(response.Content.Headers.ContentLength>25*1024*1024)return null;
    await using var input=await response.Content.ReadAsStreamAsync();using var output=new MemoryStream();var buffer=new byte[131072];int read;
    while((read=await input.ReadAsync(buffer))>0){if(output.Length+read>25*1024*1024)return null;output.Write(buffer,0,read);}
    var saved=new SavedAttachment(response.Content.Headers.ContentType?.MediaType??"application/octet-stream",Convert.ToBase64String(output.ToArray()));
    await storage.Write(key,JsonSerializer.Serialize(saved));return saved;
   }catch(HttpRequestException){ /* Use only the previously authorized copy during an outage. */ }
   catch(TaskCanceledException){ /* Use the saved copy after a network timeout. */ }
  }
  var cached=await storage.Read(key);return cached==null?null:JsonSerializer.Deserialize<SavedAttachment>(cached);
 }
 private bool mapBusy;
 private async Task DownloadMap()
 {
  if(mapBusy)return;mapBusy=true;
  try{
   status.Text="Preparing the complete recorded map for offline use…";
   using var handler=new HttpClientHandler{AllowAutoRedirect=false,CookieContainer=new CookieContainer()};
   foreach(var cookie in await terminal.CoreWebView2.CookieManager.GetCookiesAsync(AppPaths.Origin))handler.CookieContainer.Add(new Uri(AppPaths.Origin),new Cookie(cookie.Name,cookie.Value,cookie.Path));
   using var http=new HttpClient(handler){Timeout=TimeSpan.FromMinutes(10)};
   using var response=await http.GetAsync(AppPaths.Origin+"/api/mdt/offline/map-package",HttpCompletionOption.ResponseHeadersRead);
   response.EnsureSuccessStatusCode();
   var expected=response.Headers.GetValues("X-LAS-SHA256").Single();
   var temp=MapHost.MapPath+".download";long count=0;
   await using(var input=await response.Content.ReadAsStreamAsync())
   await using(var output=new FileStream(temp,FileMode.Create,FileAccess.Write,FileShare.None)){
    var buffer=new byte[131072];int read;
    while((read=await input.ReadAsync(buffer))>0){count+=read;if(count>2L*1024*1024*1024)throw new IOException("The map package exceeds the device limit.");await output.WriteAsync(buffer.AsMemory(0,read));status.Text="Downloading offline map · "+(count/1048576)+" MB";}
    output.Flush(true);
   }
   await using(var input=File.OpenRead(temp)){if(!Convert.ToHexString(await SHA256.HashDataAsync(input)).Equals(expected,StringComparison.OrdinalIgnoreCase))throw new IOException("Offline map verification failed.");}
   await maps.ReplaceMap(temp);status.Text="Offline map and road routing ready · © OpenStreetMap contributors";
   await storage.Write("map.ready",JsonSerializer.Serialize(DateTimeOffset.UtcNow.ToString("O")));
  }catch(Exception ex){status.Text="Offline map was not updated: "+ex.Message;throw;}
  finally{mapBusy=false;}
 }
 private async Task CheckUpdates(bool interactive)
 {
  try{var found=await Updates.Check();if(found?.Available==true&&Version.TryParse(found.Version,out var version)&&version>Version.Parse(Updates.Current)){release=found;updateButton.Content="Install update "+found.Version;updateButton.Visibility=Visibility.Visible;}else if(interactive)MessageBox.Show(found?.Available==true?"LAS MDT "+Updates.Current+" is up to date.":found?.Message??"Updates are temporarily unavailable.","LAS MDT updates");}
  catch(Exception ex){if(interactive)MessageBox.Show("Updates could not be checked. "+ex.Message,"LAS MDT updates");}
 }
 private async Task InstallUpdate()
 {
  if(release==null||updating)return;
  if(MessageBox.Show("Install LAS MDT "+release.Version+" and restart? Saved offline work and sign-in will be kept. Save any report you are editing first.","LAS MDT update",MessageBoxButton.YesNo,MessageBoxImage.Information)!=MessageBoxResult.Yes)return;
  updating=true;updateButton.IsEnabled=false;
  try{var path=await Updates.Download(release,text=>Dispatcher.Invoke(()=>status.Text=text));Updates.Install(path);Application.Current.Shutdown();}
  catch(Exception ex){MessageBox.Show(ex.Message,"LAS MDT update");updating=false;updateButton.IsEnabled=true;}
 }
}
