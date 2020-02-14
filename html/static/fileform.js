// save file
function save(form, path) {
  var div  = document.getElementById("log"),
      xhr  = new XMLHttpRequest();
  div.innerHTML = '';

  var fd = new FormData(form);
  var formFiles = documentFiles;
  if (formFiles != null) {
    //fd.delete('files[]');
    for(var i = 0; i < formFiles.length; i++) {
      item = formFiles[i];
      console.log('>'+item.name);
      fd.append('files[]',item);
    }
  } else {
    formFiles = form.files.files;
  }   
  if(formFiles.length === 0) {
    alert('no files');
    return false;
  }
  //console.log(...fd);

  disable_form(form, true);
  xhr.open('POST', path);
  //  xhr.setRequestHeader('Content-Type', 'application/json');
  xhr.onprogress = function (e) {
    if (e.lengthComputable) {
        console.log(e.loaded+  " / " + e.total)
    }
  }
  xhr.upload.addEventListener("progress", function(evt){
      if (evt.lengthComputable) {
        // console.log("add upload event-listener " + evt.loaded + "/" + evt.total);
        div.innerHTML = "Progress: " + evt.loaded + "/" + evt.total;
      }
    }, false);

  xhr.onloadstart = function (e) {
    console.log("start")
  }
  xhr.onloadend = function (e) {
    console.log("end")
  }

  xhr.onreadystatechange = function() {
    if (xhr.readyState != 4) return;
      console.log('Done');
    if (xhr.status != 200) {
      console.log(xhr.status + ': ' + xhr.statusText);
      div.innerHTML = xhr.statusText;
    } else {
      console.log('Result: ' + xhr.responseText);
      //rv = JSON.parse(xhr.responseText);
    
      div.innerHTML = xhr.responseText; //a.outerHTML;
    }
    disable_form(form, false);
  }
  xhr.send(fd);
  return false;
}

// code from https://gist.github.com/Peacegrove/5534309
function disable_form(form, state) {
  var elemTypes = ['input']; //, 'button', 'textarea', 'select'];
  elemTypes.forEach(function callback(type) {
    var elems = form.getElementsByTagName(type);
    disable_elements(elems, state);
  });
}

// Disables a collection of form-elements.
function disable_elements(elements, state) {
  var length = elements.length;
  while(length--) {
    elements[length].disabled = state;
  }
}


window.documentFiles = null;

function handleFileSelect(evt) {
	evt.stopPropagation();
  evt.preventDefault();
  var files = evt.target.files; // FileList object
  showFiles(files);
  documentFiles = null;//files;
  document.getElementById("log").innerHTML='';
}

function handleFileDrop(evt) {
  evt.stopPropagation();
  evt.preventDefault();
  var files = evt.dataTransfer.files; // FileList object
  showFiles(files);
  documentFiles = files;
  document.querySelector('form').reset(); // clear file input
  document.getElementById("log").innerHTML='';
}

function showFiles(files) {
  // files is a FileList of File objects. List some properties.
  var output = [];
  for (var i = 0, f; f = files[i]; i++) {
    // TODO: if (file.size > maxFileSize) {
    var h = document.createElement("strong");
    var t = document.createTextNode(f.name); // no escape needed for text
    h.appendChild(t);
    output.push('<li>',h.innerHTML,' (', f.type || 'n/a', ') - ',
                f.size, ' bytes, last modified: ',
                f.lastModifiedDate ? f.lastModifiedDate.toLocaleDateString() : 'n/a',
                '</li>');
  }
  document.getElementById('list').innerHTML = '<ul>' + output.join('') + '</ul>';
}


function handleDragOver(evt) {
  evt.stopPropagation();
  evt.preventDefault();
  evt.dataTransfer.dropEffect = 'copy'; // Explicitly show this is a copy.
}

function pageLoaded() {

  var dropZone = document.getElementById('drop_zone');

  // Check for the various File API support.
  if (window.File && window.FileReader && window.FileList && window.Blob) {
    // Setup the dnd listeners.
    dropZone.addEventListener('dragover', handleDragOver, false);
    dropZone.addEventListener('drop', handleFileDrop, false);

    document.getElementById('files').addEventListener('change', handleFileSelect, false);
  } else {
    console.log('The File APIs are not fully supported in this browser.');
    dropZone.style.display = 'none';
  }
}
