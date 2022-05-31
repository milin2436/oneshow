#!/bin/bash

sessionFile="od-se.txt"
login(){
    URL=$1
    if [ -z "$URL" ]
    then
        echo "shared url can not be empty."
        return
    fi
    rm -rf "$sessionFile"    
    wget -O /dev/null --keep-session-cookies --save-cookies "$sessionFile" "$URL"
    ret=$(grep -o 'FedAuth' $sessionFile )
    [[ "$ret" == "FedAuth" ]] && echo "login successful" || echo "Login failed"
}
dl(){
URL=$1
ODHOST=$(echo ${URL}|grep -o 'https.\{2,250\}_layouts/15/')
echo $ODHOST
tmpFile="__share_resp.html"
files="${tmpFile}.mid"
rm $tmpFile
wget --cookies=on --load-cookies=${sessionFile} -O "$tmpFile"  "${URL}"
grep -o 'UniqueId....{.\{20,45\}}' $tmpFile > $files
num=1
while read dfile
do
    #echo "$dfile id=$num"
    if [ -n "dfile" ]
    then
        st=13
        nlen=${#dfile}
        ((nlen=nlen-st-1))
        uurl="${ODHOST}download.aspx?UniqueId=${dfile:st:nlen}"
        echo "$num $uurl"
        wget  --cookies=on --load-cookies=${sessionFile} --content-disposition --restrict-file-names=nocontrol "${uurl}"
    fi
    ((num++))
done  < $files
}
if [ -z "$*" ] 
then
     echo "HELP command "
     echo "./odshare.sh login sharedURL"
     echo "./odshare.sh dl directoryURL"
     exit 0
fi

"$@"
