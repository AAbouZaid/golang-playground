# Flume metrics collector

This is a playground and here is my first `golang` code (well, not just golang, but first ever static typing language).
I didn't find Apache Flume plugin for [Telegraf](https://github.com/influxdata/telegraf), so I found that it's nice to have one.
(later I will tidy it up to work as a standard [Telegraf plugin](https://github.com/influxdata/telegraf/blob/master/CONTRIBUTING.md#input-plugin-guidelines))

The metrics source are hard codded, and you can run a simple http server to test it.
```
python -m SimpleHTTPServer
# or
python3 -m http.server 
```
**Please Note**: Flume itself returns all JSON as [strings](https://flume.apache.org/FlumeUserGuide.html#json-reporting) even numerical values.


Feel free to write comments about any part of the code.

And I have some questions you can find then as comments in the code.
