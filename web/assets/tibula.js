// PBX AI Modern Admin Controller & Tibula Script
// Copyright (C) 2007-2026 by Ubaldo Porcheddu <ubaldo@eja.it> & PBX AI Team

function tableRowCheck(obj) {
  var input = obj.querySelector('input');
  if (input) {
    input.checked = !input.checked;
  }
}

function tableRowsCheck(obj) {
  document.querySelectorAll('td').forEach(function(element) {
    var elementInput = element.querySelector('input');
    if (elementInput) {
      elementInput.checked = obj.checked;
    }
  });
}

function tableInputCheck(obj) {
  obj.checked = !obj.checked;
}

function tableEdit(obj) {
  document.querySelectorAll('td').forEach(function(element) {
    var elementInput = element.querySelector('input');
    if (elementInput) {
      elementInput.checked = false;
    }
  });
  var input = obj.querySelector('input');
  if (input) {
    input.checked = true;
  }
  document.getElementsByName('ejaAction').forEach(function(element) {
    if (element.value === 'edit') {
      element.click();
    }
  });
}

function fieldUpload(name) {
  var el = window._protected_reference = document.createElement('INPUT');
  el.type = 'file';
  el.addEventListener('change', function(ev) {
    var input = ev.target;
    var reader = new FileReader();
    reader.onload = function() {
      if (document.forms[0] && document.forms[0].elements['ejaValues[' + name + ']']) {
        document.forms[0].elements['ejaValues[' + name + ']'].value = reader.result;
      }
    };
    reader.readAsText(input.files[0]);
  });
  el.click();
}

function fieldDownload(o, name) {
  var fileName = prompt('Save As');
  if (fileName) {
    o.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(document.forms[0].elements['ejaValues[' + name + ']'].value));
    o.setAttribute('download', fileName);
    return true;
  } else {
    return false;
  }
}

function fieldEditor(name) {
  var o = document.getElementsByName('ejaValues[' + name + ']')[0];
  if (!editors.hasOwnProperty(name) && window.SUNEDITOR) {
    editors[name] = SUNEDITOR.create(o, {
      buttonList: [
        ['fullScreen', 'undo', 'redo'],
        ['font', 'fontSize', 'formatBlock'],
        ['paragraphStyle', 'blockquote'],
        ['bold', 'underline', 'italic', 'strike', 'subscript', 'superscript'],
        ['fontColor', 'hiliteColor', 'textStyle'],
        ['removeFormat'],
        ['outdent', 'indent'],
        ['align', 'horizontalRule', 'list', 'lineHeight'],
        ['table', 'link', 'image', 'video', 'audio'],
        ['showBlocks', 'codeView', 'print']
      ]
    });
  }
}

function fieldCalendar(name) {
  var iframeHtml = '<iframe src="/static/calendar.html?id=' + name + '" class="w-100 h-100 border-0" style="min-height:100vh"></iframe>';
  var closeBtn = '<button class="btn-close position-fixed top-0 end-0 m-3" onclick="this.closest(\'.modal\').remove()"></button>';
  var modalHtml = '<div class="modal fade show d-block"><div class="modal-dialog modal-fullscreen"><div class="modal-content bg-white">' + iframeHtml + closeBtn + '</div></div></div>';
  document.body.insertAdjacentHTML('beforeend', modalHtml);
}

function formInit() {
  var f = document.getElementById('ejaForm');
  if (!f) return;
  var o = {};
  
  Array.from(f.elements).forEach(function(i) {
    o[i.name] = i.value;
  });
  
  f.oninput = function() {
    window.onbeforeunload = function() {
      return 'Unsaved changes';
    };
  };

  f.onsubmit = function() {
    window.onbeforeunload = null;
  };
  
  window.onbeforeunload = function() {
    return null;
  };
}

// Enhance Navbar with Direct Dashboard and Chat Links
function enhanceNavbar() {
  var nav = document.querySelector('.navbar .container-fluid');
  if (!nav) return;

  if (document.getElementById('pbx-quick-links')) return;

  var quickContainer = document.createElement('div');
  quickContainer.id = 'pbx-quick-links';
  quickContainer.className = 'd-flex align-items-center gap-2 ms-auto me-3';

  quickContainer.innerHTML = `
    <a href="/dashboard" class="pbx-nav-badge pbx-nav-badge-dash" title="Operations & Telephony Dashboard">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
      <span>Dashboard</span>
    </a>
    <a href="/chat" class="pbx-nav-badge pbx-nav-badge-chat" target="_blank" title="Open AI Chat Assistant">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
      <span>Chat AI</span>
    </a>
  `;

  // Insert before the last button or at end
  var lastAction = nav.querySelector('.btn-group, .btn-danger, .btn-secondary');
  if (lastAction) {
    nav.insertBefore(quickContainer, lastAction);
  } else {
    nav.appendChild(quickContainer);
  }
}

var editors = [];

document.addEventListener('DOMContentLoaded', function() {
  document.querySelectorAll('.toast').forEach(function(toast) {
    if (window.bootstrap && bootstrap.Toast) {
      var toastInstance = new bootstrap.Toast(toast);
      setTimeout(function() {
        toastInstance.hide();
      }, 5000);
    }
  });

  document.querySelectorAll('select[multiple]').forEach(function(select) {
    if (window.SlimSelect) {
      new SlimSelect({
        select: select,
        settings: {
          placeholderText: ''
        }
      });
    }
  });

  enhanceNavbar();
});

var ejaForm = document.getElementById('ejaForm');
if (ejaForm) {
  ejaForm.addEventListener('submit', function() {
    this.querySelectorAll('select').forEach(function(select) {
      if (select.selectedIndex === -1 || select.value === '') {
        select.value = '';
      }
    });
    for (var key in editors) {
      if (editors[key] && editors[key].save) {
        editors[key].save();
      }
    }
  });
}

window.onload = function() {
  if (document.getElementsByName('ejaGoogleSsoId').length > 0 && window.google) {
    google.accounts.id.initialize({
      client_id: document.getElementsByName('ejaGoogleSsoId')[0].value,
      callback: function(e) {
        document.getElementsByName('ejaValues[googleSsoToken]')[0].value = e.credential;
        document.getElementById('ejaForm').submit();
      }
    });
    google.accounts.id.renderButton(document.getElementById("google"), { type: "icon" });
  }

  // Auto refresh session warning
  setTimeout(function() {
    console.info("PBX AI Session active");
  }, 9700 * 1000);

  formInit();
  enhanceNavbar();
};
