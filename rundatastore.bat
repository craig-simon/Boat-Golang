rem install from https://dl.google.com/dl/cloudsdk/channels/rapid/GoogleCloudSDKInstaller.exe
powershell -Command "gcloud beta emulators datastore start --data-dir db --host-port localhost:8169"
pause
