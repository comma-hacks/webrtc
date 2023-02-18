#!/bin/bash
#
# Start (or restart) the camera pipeline on the tici.
# It will start the daemons as windows on the comma tmux session
# Avoids using the ignition just to use the camera pipeline

ssh -S /tmp/ssh-master-tici -M -N -f tici
ssh -S /tmp/ssh-master-tici -O check tici

cmds=(
  "camerad:/data/openpilot/system/camerad:./camerad"
  "encoderd:/data/openpilot/selfdrive/loggerd:./encoderd"
  "bridge:/data/openpilot/cereal/messaging:./bridge"
)

for cmd in "${cmds[@]}"; do
  name=$(echo "$cmd" | cut -d: -f1)
  dir=$(echo "$cmd" | cut -d: -f2)
  prog=$(echo "$cmd" | cut -d: -f3)

  echo $name $dir $prog
  ssh -S /tmp/ssh-master-tici tici pkill $name
  while ssh -S /tmp/ssh-master-tici tici pgrep -f $prog; do
    echo "still waiting for $name to close"
    sleep 0.2
  done
  ssh -S /tmp/ssh-master-tici tici tmux new-window -t comma -n $name -c $dir $prog
done

ssh -S /tmp/ssh-master-tici -O exit tici
echo "Commands executed."