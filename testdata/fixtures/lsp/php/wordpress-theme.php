<?php
/**
 * Template Name: Full Width
 * Description: A WordPress page template using design tokens
 */
$site_name = get_bloginfo('name');
$show_posts = have_posts();
?>
<!DOCTYPE html>
<html <?php language_attributes(); ?>>
<head>
<style>
:root {
  --color-primary: #0073aa;
}
.site-header {
  background-color: var(--color-primary);
  padding: var(--spacing-lg);
}
</style>
</head>
<body <?php body_class(); ?>>
  <header class="site-header">
    <h1 style="color: var(--color-text); font-size: var(--font-size-xl)"><?php echo esc_html($site_name); ?></h1>
  </header>

  <main>
    <?php
    if ($show_posts) :
      while (have_posts()) : the_post();
    ?>
    <article style="margin: var(--spacing-md)">
      <h2><?php the_title(); ?></h2>
      <?php the_content(); ?>
    </article>
    <?php
      endwhile;
    endif;
    ?>
  </main>

  <style>
    .site-footer {
      border-top: 1px solid var(--color-border);
      padding: var(--spacing-sm);
    }
  </style>

  <footer class="site-footer">
    <p>&copy; <?php echo esc_html($site_name); ?></p>
  </footer>
</body>
</html>
