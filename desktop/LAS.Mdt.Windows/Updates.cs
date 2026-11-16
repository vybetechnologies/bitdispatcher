using System.Diagnostics;
using System.IO;
using System.Net.Http;
using System.Reflection;
using System.Security.Cryptography;
using System.Text.Json;

namespace LasMdt;
internal sealed record Release(bool Available,string Version,string URL,string SHA256,long Size,string Message);
internal static class Updates
{
 internal static string Current=>Assembly.GetExecutingAssembly().GetName().Version?.ToString(3)??"1.0.0";
 internal static bool TrustedDownload(Uri uri)=>uri.Scheme=="https"&&uri.Host=="github.com"&&uri.AbsolutePath.StartsWith("/vybetechnologies/bitdispatcher/releases/download/mdt-v",StringComparison.Ordinal)&&uri.AbsolutePath.EndsWith("/LAS-MDT-Setup-x64.exe",StringComparison.Ordinal);
 internal static async Task<Release?> Check()
 {
  using var http=new HttpClient{Timeout=TimeSpan.FromSeconds(15)};
  var json=await http.GetStringAsync(AppPaths.Origin+"/api/mdt/windows-release");
  return JsonSerializer.Deserialize<Release>(json,new JsonSerializerOptions{PropertyNameCaseInsensitive=true});
 }
 internal static async Task<string> Download(Release release,Action<string> progress)
 {
  if(!release.Available||!Version.TryParse(release.Version,out var version)||version<=Version.Parse(Current)||!TrustedDownload(new Uri(release.URL))||release.SHA256.Length!=64||release.Size<100_000||release.Size>400*1024*1024)throw new InvalidOperationException("The update manifest is invalid.");
  using var handler=new HttpClientHandler{AllowAutoRedirect=false};
  using var http=new HttpClient(handler){Timeout=TimeSpan.FromMinutes(15)};
  var uri=new Uri(release.URL);HttpResponseMessage? response=null;
  for(int redirects=0;redirects<5;redirects++){
   response=await http.GetAsync(uri,HttpCompletionOption.ResponseHeadersRead);
   if((int)response.StatusCode<300||(int)response.StatusCode>=400)break;
   var next=response.Headers.Location;if(next==null)throw new InvalidOperationException("The update download redirect is invalid.");
   uri=next.IsAbsoluteUri?next:new Uri(uri,next);
   if(uri.Scheme!="https"||!(uri.Host=="github.com"||uri.Host.EndsWith(".githubusercontent.com",StringComparison.Ordinal)))throw new InvalidOperationException("Untrusted update download host.");
   response.Dispose();
  }
  if(response==null)throw new InvalidOperationException("The update could not be downloaded.");
  using(response){
   response.EnsureSuccessStatusCode();
   var dir=Path.Combine(AppPaths.Data,"updates");Directory.CreateDirectory(dir);
   var path=Path.Combine(dir,"LAS-MDT-"+release.Version+".exe");
   await using(var output=new FileStream(path+".download",FileMode.Create,FileAccess.Write,FileShare.None)){
    await using var input=await response.Content.ReadAsStreamAsync();var buffer=new byte[131072];long count=0;int read;
    while((read=await input.ReadAsync(buffer))>0){count+=read;if(count>release.Size)throw new InvalidOperationException("Update size exceeded its manifest.");await output.WriteAsync(buffer.AsMemory(0,read));progress("Downloading update "+(100*count/release.Size)+"%");}
    output.Flush(true);if(count!=release.Size)throw new InvalidOperationException("Incomplete update download.");
   }
   await using(var input=File.OpenRead(path+".download")){var digest=Convert.ToHexString(await SHA256.HashDataAsync(input));if(!digest.Equals(release.SHA256,StringComparison.OrdinalIgnoreCase))throw new InvalidOperationException("Update integrity verification failed.");}
   File.Move(path+".download",path,true);return path;
  }
 }
 internal static void Install(string path)=>Process.Start(new ProcessStartInfo(path,"/S /UPDATE"){UseShellExecute=true});
}
