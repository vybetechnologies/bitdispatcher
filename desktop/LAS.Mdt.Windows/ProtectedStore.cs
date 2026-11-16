using System.IO;
using System.Security.Cryptography;
using System.Text;

namespace LasMdt;
internal sealed class ProtectedStore
{
 private readonly string folder;
 private readonly SemaphoreSlim gate=new(1,1);
 internal ProtectedStore(string folder){this.folder=folder;Directory.CreateDirectory(folder);}
 private string PathFor(string key)
 {
  if(key.Length>600)throw new InvalidOperationException("Storage key is too long.");
  return Path.Combine(folder,Convert.ToHexString(SHA256.HashData(Encoding.UTF8.GetBytes(key)))+".dat");
 }
 internal async Task<string?> Read(string key)
 {
  await gate.WaitAsync();
  try {var path=PathFor(key);if(!File.Exists(path))return null;return Encoding.UTF8.GetString(ProtectedData.Unprotect(await File.ReadAllBytesAsync(path),null,DataProtectionScope.CurrentUser));}
  finally{gate.Release();}
 }
 internal async Task Write(string key,string value)
 {
  if(Encoding.UTF8.GetByteCount(value)>48*1024*1024)throw new InvalidOperationException("This entry exceeds the offline storage limit. Keep the source file and reduce its size.");
  await gate.WaitAsync();
  try {
   var path=PathFor(key);var encrypted=ProtectedData.Protect(Encoding.UTF8.GetBytes(value),null,DataProtectionScope.CurrentUser);
   await using(var f=new FileStream(path+".tmp",FileMode.Create,FileAccess.Write,FileShare.None)){await f.WriteAsync(encrypted);f.Flush(true);}
   File.Move(path+".tmp",path,true);
  }finally{gate.Release();}
 }
 internal async Task Delete(string key){await gate.WaitAsync();try{File.Delete(PathFor(key));}finally{gate.Release();}}
 internal bool Exists(string key)=>File.Exists(PathFor(key));
}
internal static class NativeSelfTest
{
 internal static void Run()
 {
  var path=Path.Combine(Path.GetTempPath(),"las-mdt-test-"+Guid.NewGuid());var store=new ProtectedStore(path);
  const string value="synthetic offline note";
  store.Write("outbox",value).GetAwaiter().GetResult();
  if(new ProtectedStore(path).Read("outbox").GetAwaiter().GetResult()!=value)throw new Exception("Offline data did not survive restart.");
  if(Directory.GetFiles(path).Any(p=>File.ReadAllText(p).Contains(value)))throw new Exception("Offline storage was not protected.");
  store.Delete("outbox").GetAwaiter().GetResult();if(store.Read("outbox").GetAwaiter().GetResult()!=null)throw new Exception("Delete failed.");
  Directory.Delete(path);
  using(var compressed=new MemoryStream()){
   using(var zip=new System.IO.Compression.GZipStream(compressed,System.IO.Compression.CompressionMode.Compress,true))zip.Write(new byte[]{26,3,10,1,65});
   var decoded=MapHost.ResponseBytes(new MapAnswer(200,"application/vnd.mapbox-vector-tile","gzip",Convert.ToBase64String(compressed.ToArray())));
   if(!decoded.SequenceEqual(new byte[]{26,3,10,1,65}))throw new Exception("WebView map response must be decoded protobuf, not gzip bytes.");
  }
  using(var map=new MapHost()){var glyph=map.Request("/api/map/fonts/Noto%20Sans%20Regular/0-255.pbf",null).GetAwaiter().GetResult();if(glyph.Status!=200||string.IsNullOrEmpty(glyph.Data))throw new Exception("Offline map engine IPC failed.");}
  if(!Updates.TrustedDownload(new Uri("https://github.com/vybetechnologies/bitdispatcher/releases/download/mdt-v1.0.0/LAS-MDT-Setup-x64.exe")))throw new Exception("Trusted release rejected.");
  if(Updates.TrustedDownload(new Uri("https://example.com/installer.exe")))throw new Exception("Untrusted updater host accepted.");
  foreach(var navigationPath in new[]{"/#/mdt","/#/mdt/reports","/#/mdt/reports/OFFLINE-test","/"})if(!TerminalWindow.TerminalPage(AppPaths.Origin+navigationPath))throw new Exception("MDT navigation rejected.");
  foreach(var navigationPath in new[]{"/#/admin","/#/console","/#/taker","/#/rms","/api/state","/#/mdt-other"})if(TerminalWindow.TerminalPage(AppPaths.Origin+navigationPath))throw new Exception("Another module was allowed in the Windows MDT.");
 }
}
