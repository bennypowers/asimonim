<?php
/**
 * WordPress template partial for testing design token completion.
 *
 * The CSS below intentionally uses incomplete custom property references
 * (e.g. "--bg" instead of "var(--bg)") to simulate the cursor position
 * where the LSP completion provider should trigger and offer var() wrapping.
 */
$nav_label = get_bloginfo('name');
?>
<style>
.site-nav {
  background: --bg;
  color: --color;
}
</style>
<main>
  <nav aria-label="<?php echo esc_attr($nav_label); ?>">
    <?php wp_nav_menu(); ?>
  </nav>
</main>
