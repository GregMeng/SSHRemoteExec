# SSHRemoteExec
Batch execute a script on the remote host by ssh

通过 ssh 在远程主机上批量执行脚本

1. 从 Conf/all.login 中读取远程主机 IP，账号，密码。

2. 将 Script 目录下的脚本通过 scp 上传到远程主机，
   为了防止脚本在传输过程中发生改变，需要对比传输前后的md5值。

3. 对远程主机先 ping 一次，如果不通直接返回 ping error。

4. 用goroutine并发执行。

5. 将执行结果通过管道返回后写入本地 redis。

6. redis 中以 hset 方式存，key 是 taskid，taskid 从 1 开始递增。
