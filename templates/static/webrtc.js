function hasGetUserMedia() {
    return !!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia);
  }
  if (hasGetUserMedia()) {
    // Good to go!
  } else {
    alert("getUserMedia() is not supported by your browser");
  }
  
const constraints = {
  video: true,
};


  /*
const hdConstraints = {
    video: { width: { min: 1280 }, height: { min: 720 } },
  };

  navigator.mediaDevices.getUserMedia(hdConstraints).then((stream) => {
    video.srcObject = stream;
  });
  
const vgaConstraints = {
    video: { width: { exact: 640 }, height: { exact: 480 } },
  };
  
navigator.mediaDevices.getUserMedia(vgaConstraints).then((stream) => {
    video.srcObject = stream;
  });
*/

const video = document.querySelector("video");

navigator.mediaDevices.getUserMedia(constraints).then((stream) => {
  video.srcObject = stream;
});
