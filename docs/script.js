
    // Highlight active sidebar link on scroll
    const sections = document.querySelectorAll('section');
    const navLinks = document.querySelectorAll('.sidebar nav a');
    const menuToggle = document.querySelector('.menu-toggle');
    const sidebarNav = document.querySelector('.sidebar-nav');

    menuToggle.addEventListener('click', () => {
      sidebarNav.classList.toggle('active');
    });

    // Highlight active sidebar link on scroll
    window.addEventListener('scroll', () => {
      let current = '';
      sections.forEach(section => {
        const sectionTop = section.offsetTop - 60;
        if (pageYOffset >= sectionTop) {
          current = section.getAttribute('id');
        }
      });
      navLinks.forEach(link => {
        link.classList.remove('active');
        if (link.getAttribute('href') === '#' + current) {
          link.classList.add('active');
        }
      });
    });

    // Close sidebar when a link is clicked on mobile
    navLinks.forEach(link => {
      link.addEventListener('click', () => {
        if (window.innerWidth <= 900) { // Assuming 900px is our mobile breakpoint
          sidebarNav.classList.remove('active');
        }
      });
    });
