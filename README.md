# SSHRemoteExec
Batch execute a script on the remote host by ssh

通过 ssh 在远程主机上批量执行脚本

1. 从Conf/all.login中读取远程主机IP，账号，密码

2. 将Script目录下的脚本通过scp上传到远程主机，
   为了防止脚本在传输过程中发生改变，需要对比传输前后的md5值。

3. 对远程主机先ping一次，如果不同直接，写日志ping error。

4. 用goroutine并发执行。

5. 将执行结果通过管道返回存入本地redis。

6. redis中以hset方式存，key是taskid，taskid从1开始递增。
