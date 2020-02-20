// save file
var Meta = {
    token: '',        // session token
    timer: null,      // keepalive timer
    timeout: 5000,    // ping & reconnect timeout
};

function sendForm(form, path) {
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

  xhr.open('POST', path);
  //xhr.responseType = 'json';
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
      rv = JSON.parse(xhr.responseText);
      for (let [key, value] of Object.entries(rv.files)) {
        console.log(`${key}: ${value}`);
        var elem = document.querySelectorAll("[data-filename='"+key+"']")[0];
        elem.dataset.fileid = value;
        elem.innerHTML='Received';
      }
      div.innerHTML = 'Done';//xhr.responseText; //a.outerHTML;
    }
    disable_form(form, false);
  }
  document.getElementById("abort").addEventListener("click", 
   function() {
    console.log('Aborted by user');
    xhr.abort();
  });
  disable_form(form, true);

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
    var e = elements[length];
   if (e.classList.contains('reversed')) {
    e.disabled = !state;
   } else {
    e.disabled = state;
   }
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

function cell(val,className) {
  var c = document.createElement("div");
  if (className === undefined) className = "Rtable-short";
  c.classList.add("Rtable-cell");
  c.classList.add(className);
  var t = document.createTextNode(val); 
  c.appendChild(t);
  return c
}
function cellLink(val,id) {
  if (id === undefined) return cell(val,"Rtable-long");
  var c = document.createElement("div");
  c.classList.add("Rtable-cell");
  c.classList.add("Rtable-long");
  var a = document.createElement('a');
  var link = document.createTextNode(val); 
  a.appendChild(link); 
  a.href = '/file/'+id;
  c.appendChild(a);
  return c
}
function row(elem,name,ctype,size,time,state,id) {
  var c = document.createElement("div");
  c.classList.add("row");
  c.appendChild(cellLink(name,id));
  c.appendChild(cell(ctype));
  c.appendChild(cell(size));
  c.appendChild(cell(time));
  c.appendChild(cell(state));
  var cmd = cell('[x]');
  cmd.dataset.filename = name;
  c.appendChild(cmd);
  elem.appendChild(c);
}

function showFiles(files) {
  // files is a FileList of File objects. List some properties.
  var d= document.getElementById('list');
  d.innerHTML='';
  for (var i = 0, f; f = files[i]; i++) {
    // TODO: if (file.size > maxFileSize) {
       row(d,
        f.name, 
        f.type || 'n/a', 
        f.size/1000 + 'Kb',
        f.lastModifiedDate ? f.lastModifiedDate.toLocaleDateString() : 'n/a',
        'Ready'
      );
  }
}

function handleDragOver(evt) {
  evt.stopPropagation();
  evt.preventDefault();
  evt.dataTransfer.dropEffect = 'copy'; // Explicitly show this is a copy.
}


function clearForm(form) {
  console.log('reset');
  documentFiles = null;
  document.getElementById('list').innerHTML = '';
  document.querySelector('form').reset(); // clear file input
  document.getElementById("log").innerHTML='';

  return true;
}

function pageLoaded(useFiles) {
  if (useFiles) {
    getFiles();
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
  getProfile()
}

function getFiles() {
  var xhr  = new XMLHttpRequest();
  xhr.open('GET', '/api/files');
  xhr.onreadystatechange = function() {
    if (xhr.readyState != 4) return;
    if (xhr.status != 200) {
      console.log(xhr.status + ': ' + xhr.statusText);
    } else {
      var files = JSON.parse(xhr.responseText);
      if (files == undefined) return;
      var d = document.getElementById('stored');
      d.innerHTML='';
      for (var i = 0, f; f = files[i]; i++) {
        // TODO: if (file.size > maxFileSize) {
           row(d,
            f.name, 
            f.type || 'n/a', 
            f.size/1000 + 'Kb',
            f.created_at ? new Date(f.created_at).toLocaleDateString() : 'n/a',
            f.state,
            f.id
          );
      }
    }
  }
  xhr.send();
}

function getProfile() {
  var xhr  = new XMLHttpRequest();
  xhr.open('GET', '/api/profile');
  xhr.onreadystatechange = function() {
    if (xhr.readyState != 4) return;
    if (xhr.status != 200) {
      console.log(xhr.status + ': ' + xhr.statusText);
    } else {
      var profile = JSON.parse(xhr.responseText);
      if (profile == undefined) return;
      Meta.token = profile.token;
      openStream();
    }
  }
  xhr.send();
}

function openAgain() {
  if (Meta.timer) {
    clearTimeout(Meta.timer);
  }
  Meta.timer = setTimeout(openStream, Meta.timeout);
} 

// Setup websocket
function openStream() {
  try {
    var loc = window.location, url;
    if (loc.protocol === "https:") {
        url = "wss:";
    } else {
        url = "ws:";
    }
    url += "//" + loc.host;
    url += "/" + "ws";
    if (typeof RequestID === 'undefined') {
      url += '/-'
    } else {
      url += '/'+RequestID
    }
    url += '/'+Meta.token+'/';

    c = new WebSocket(url);
    if (c == null) {
      openAgain() // TODO: do we need it?
      return
    }
    var d = document.getElementById('stream');
    console.log("Token: "+Meta.token)
/*
    send = function(data){
      if (c != null) c.send(data)
    }
*/
    c.onopen = function(){
      console.log("WS OPEN")
      document.getElementById("log").innerHTML='';
     }
    c.onerror = function(evt) {
      console.log("WS ERROR: " + evt.data)
    }
    c.onclose = function(evt) {
      console.log("WS CLOSE")
      c = null;
      if (event.wasClean) {
        console.debug('Connection closed clean');
      } else {
        console.debug('Connection aborted');
      }
      console.debug('Code: ' + event.code + ' reason: ' + event.reason);
      document.getElementById("log").innerHTML='Connection closed';
      openAgain()
    }
    c.onmessage = function(msg) {
      console.log('WS>>>'+msg.data)
      var m = JSON.parse(msg.data);
      if (m.type == "widget") {
        console.log('include ' + m.id)
        document.getElementById(m.id).innerHTML = m.data;
      } else if (m.type == "file") {
        if (m.state == 'saved'){
          var elem = document.querySelectorAll("[data-fileid='"+m.id+"']")[0];
          console.log('ELEM4RM',elem)
          if (elem != undefined) {
           elem.parentElement.remove();
          }
          getFiles();
        } else {
          console.log('Unhandled message: '+msg.data)
        }
      }
    }
  } catch(e){
    console.log(e);
  }
}