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

// Enhance Navbar with Direct Config, Dashboard and Chat Links
function enhanceNavbar() {
  var nav = document.querySelector('.navbar .container-fluid');
  if (!nav) return;

  if (document.getElementById('pbx-quick-links')) return;

  var sessionInput = document.querySelector('input[name="ejaSession"]');
  var session = sessionInput ? sessionInput.value : '';

  var quickContainer = document.createElement('div');
  quickContainer.id = 'pbx-quick-links';
  quickContainer.className = 'd-flex align-items-center gap-2 ms-auto me-3';

  var configBadge = '';
  if (session) {
    configBadge = `
      <a href="?ejaSession=${session}&ejaModuleId=29" class="pbx-nav-badge pbx-nav-badge-cfg" title="Configuración de Modelos e IA">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
        <span>⚙️ Configuración AI</span>
      </a>
    `;
  }

  quickContainer.innerHTML = `
    ${configBadge}
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

// Render modern Home Launcher Grid on root page
function initHomeLauncher() {
  var sessionInput = document.querySelector('input[name="ejaSession"]');
  var moduleIdInput = document.querySelector('input[name="ejaModuleId"]');
  if (!sessionInput || !moduleIdInput) return;

  var session = sessionInput.value;
  var moduleId = moduleIdInput.value;

  // Only trigger on root/home page (ModuleId 1) when not in a subform
  if (moduleId !== "1") return;

  var targetRow = document.querySelector('.container-fluid > .row');
  if (!targetRow || document.getElementById('pbx-launcher-root')) return;

  var launcher = document.createElement('div');
  launcher.id = 'pbx-launcher-root';
  launcher.className = 'pbx-home-launcher w-100';

  launcher.innerHTML = `
    <div class="pbx-hero-banner">
      <div class="pbx-hero-title">
        <span>🤖 Centro de Control PBX AI</span>
      </div>
      <p class="pbx-hero-sub">
        Bienvenido al panel central de administración. Selecciona un módulo para gestionar modelos LLM, prompts de voz, extensiones Asterisk, cuentas SIP y auditoría.
      </p>
    </div>

    <div class="pbx-launcher-grid">
      <!-- 1. AI Settings -->
      <a href="?ejaSession=${session}&ejaModuleId=29" class="pbx-launcher-card">
        <div class="pbx-card-icon pbx-icon-purple">⚙️</div>
        <div class="pbx-card-body">
          <h4>Configuración AI</h4>
          <p>Modelos LLM (OpenAI, Claude, Ollama, Groq), tokens de API, endpoints y proveedores ASR/TTS.</p>
        </div>
      </a>

      <!-- 2. AI Prompts -->
      <a href="?ejaSession=${session}&ejaModuleId=28" class="pbx-launcher-card">
        <div class="pbx-card-icon pbx-icon-blue">📝</div>
        <div class="pbx-card-body">
          <h4>Prompts del Sistema</h4>
          <p>Personaliza las instrucciones de comportamiento para llamadas telefónicas (Asterisk), chat web y canal global.</p>
        </div>
      </a>

      <!-- 3. AI SIP -->
      <a href="?ejaSession=${session}&ejaModuleId=30" class="pbx-launcher-card">
        <div class="pbx-card-icon pbx-icon-green">📞</div>
        <div class="pbx-card-body">
          <h4>Cuentas SIP / Asterisk</h4>
          <p>Gestión de extensiones, troncales y sincronización de credenciales PJSIP en tiempo real.</p>
        </div>
      </a>

      <!-- 4. AI Users -->
      <a href="?ejaSession=${session}&ejaModuleId=32" class="pbx-launcher-card">
        <div class="pbx-card-icon pbx-icon-cyan">👥</div>
        <div class="pbx-card-body">
          <h4>Usuarios Autorizados</h4>
          <p>Control de acceso telefónico mediante PIN, números autorizados y perfiles de usuario.</p>
        </div>
      </a>

      <!-- 5. AI Actions -->
      <a href="?ejaSession=${session}&ejaModuleId=24" class="pbx-launcher-card">
        <div class="pbx-card-icon pbx-icon-amber">⚡</div>
        <div class="pbx-card-body">
          <h4>Acciones Rápidas</h4>
          <p>Comandos rápidos predefinidos del bot (/welcome, /help, etc.) para atención inmediata.</p>
        </div>
      </a>

      <!-- 6. AI Languages & Translations -->
      <a href="?ejaSession=${session}&ejaModuleId=26" class="pbx-launcher-card">
        <div class="pbx-card-icon pbx-icon-purple">🌍</div>
        <div class="pbx-card-body">
          <h4>Idiomas & Traducciones</h4>
          <p>Configuración de múltiples idiomas, diccionarios y cadenas de respuesta traducibles.</p>
        </div>
      </a>

      <!-- 7. AI Logs -->
      <a href="?ejaSession=${session}&ejaModuleId=27" class="pbx-launcher-card">
        <div class="pbx-card-icon pbx-icon-blue">📜</div>
        <div class="pbx-card-body">
          <h4>Registros & Logs</h4>
          <p>Auditoría de eventos, latencias de modelos, consumo de tokens y llamadas registradas.</p>
        </div>
      </a>

      <!-- 8. Real-time Dashboard -->
      <a href="/dashboard" class="pbx-launcher-card" style="border-color: rgba(99, 102, 241, 0.4);">
        <div class="pbx-card-icon pbx-icon-blue">📊</div>
        <div class="pbx-card-body">
          <h4>Dashboard de Operaciones</h4>
          <p>Analítica en vivo, métricas de latencia, tasa de errores y visor interactivo de diálogos.</p>
        </div>
      </a>

      <!-- 9. Web Chat Assistant -->
      <a href="/chat" target="_blank" class="pbx-launcher-card" style="border-color: rgba(6, 182, 212, 0.4);">
        <div class="pbx-card-icon pbx-icon-cyan">💬</div>
        <div class="pbx-card-body">
          <h4>Chat Web Asistente</h4>
          <p>Interactúa y prueba el asistente inteligente con soporte para voz en tiempo real y subida de archivos.</p>
        </div>
      </a>
    </div>
  `;

  targetRow.appendChild(launcher);
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
  initHomeLauncher();
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
  initHomeLauncher();
};
