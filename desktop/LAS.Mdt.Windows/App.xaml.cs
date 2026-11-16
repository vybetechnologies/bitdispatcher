using System.IO;
using System.Windows;

namespace LasMdt;
public partial class App : Application
{
 private FileStream? instance;
 protected override void OnStartup(StartupEventArgs e)
 {
  base.OnStartup(e);
  if(e.Args.Contains("--webview-self-test")){NativeWebViewTest.Start();return;}
  if (e.Args.Contains("--self-test")) {try {Task.Run(NativeSelfTest.Run).GetAwaiter().GetResult();Shutdown(0);}catch(Exception ex){File.WriteAllText("native-test-error.txt",ex.ToString());Shutdown(1);}return;}
  try
  {
   Directory.CreateDirectory(AppPaths.Data);
   instance=new FileStream(Path.Combine(AppPaths.Data,"instance.lock"),FileMode.OpenOrCreate,FileAccess.ReadWrite,FileShare.None);
   MainWindow=new TerminalWindow();MainWindow.Show();
  }
  catch(IOException) {MessageBox.Show("LAS MDT is already open for this Windows account.","Lakes Area Security MDT");Shutdown();}
  catch(Exception ex) {MessageBox.Show("LAS MDT could not start. "+ex.Message,"Lakes Area Security MDT");Shutdown(1);}
 }
 protected override void OnExit(ExitEventArgs e){instance?.Dispose();base.OnExit(e);}
}
internal static class AppPaths
{
 internal static string Data=>Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),"Lakes Area Security","MDT");
 internal const string Origin="https://cad.lakesareasecurity.com";
}
