# ascii-art

TODO: make this code... good?

```sh
mkdir images
wget https://daverupert.com/images/posts/2022/spectrum/color-wheel.png -O images/color_wheel.png
go run . -f images/rainbow.png -w 100 -i
go run . -f images/color_wheel.png -w 100 -c -i
```