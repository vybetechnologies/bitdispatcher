using System.Diagnostics;
using System.IO;
using System.Text.Json;
using System.Text;
namespace LasMdt;
internal sealed record MapAnswer(int Status,string ContentType,string? Encoding,string? Data);
internal sealed class MapHost : IDisposable
{
 internal static string MapPath=>Path.Combine(AppPaths.Data,"recorded-map.db");
 private Process? worker;
 private readonly SemaphoreSlim gate=new(1,1);
 internal async Task<MapAnswer> Request(string path,string? body)
 {
  await gate.WaitAsync();
  try{
   if(worker==null||worker.HasExited){
    worker=Process.Start(new ProcessStartInfo(Path.Combine(AppContext.BaseDirectory,"LAS.MDT.Map.exe"),File.Exists(MapPath)?"\""+MapPath+"\"":""){UseShellExecute=false,CreateNoWindow=true,RedirectStandardInput=true,RedirectStandardOutput=true,StandardInputEncoding=new UTF8Encoding(false),StandardOutputEncoding=Encoding.UTF8});
    if(worker==null)throw new IOException("Offline map engine did not start.");
   }
   using var data=JsonDocument.Parse(body??"null");
   await worker.StandardInput.WriteLineAsync(JsonSerializer.Serialize(new{Path=path,Body=data.RootElement}));
   await worker.StandardInput.FlushAsync();
   var line=await worker.StandardOutput.ReadLineAsync().WaitAsync(TimeSpan.FromSeconds(45));
   return JsonSerializer.Deserialize<MapAnswer>(line??throw new IOException("Offline map engine stopped."))??throw new IOException("Invalid offline map reply.");
  }catch{Stop();throw;}finally{gate.Release();}
 }
 internal async Task ReplaceMap(string temp){await gate.WaitAsync();try{Stop();File.Delete(MapPath+"-wal");File.Delete(MapPath+"-shm");File.Move(temp,MapPath,true);}finally{gate.Release();}}
 private void Stop(){if(worker!=null){if(!worker.HasExited){worker.StandardInput.Close();if(!worker.WaitForExit(5000))worker.Kill();}worker.WaitForExit(5000);worker.Dispose();worker=null;}}
 public void Dispose()=>Stop();
}
