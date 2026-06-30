# 自定义字体目录

把你要加入 Collabora 的字体文件（.ttf / .ttc / .otf）直接放进这个目录，
然后重建并重启 Collabora：

    docker compose build collabora
    docker compose up -d collabora

构建时这些文件会被 COPY 到容器的 /usr/share/fonts/custom/ 并执行 fc-cache。
验证字体是否生效（把 "你的字体名" 换成 fc-list 里显示的名字）：

    docker compose exec collabora fc-list | grep -i 你的字体名

注意：商业字体（如 Windows 的宋体 SimSun、微软雅黑）请确认你有合法使用授权。
