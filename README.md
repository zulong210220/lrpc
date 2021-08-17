## lrpc

自制rpc框架

- 编码解码


fix 
- 数据竞争panic
- read阻塞,或者全部读取解析出错


粘包 导致阻塞

```
' {"MagicNumber":3927900,"CodecType":"application/json","ConnectTimeout":3000000000,"HandleTimeout":0}Foo.Sum{"Num1":2,"Num2":4} ' 146 <nil> 1024
```
